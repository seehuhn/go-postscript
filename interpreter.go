// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
	"io"
	"maps"
	"slices"
	"strings"

	"seehuhn.de/go/membudget"
)

// Interpreter represents one instance of the PostScript interpreter.
type Interpreter struct {
	// CheckStart can be set to true to make the interpreter check that the
	// first two input bytes are "%!"; otherwise, ErrNoPostScript is returned.
	// This flag must be set before the first call to Execute.
	CheckStart bool

	// MaxOps can be set to a positive value to limit the number of executed
	// operations.  Operators which process many elements or bytes count as
	// several operations.  If this limit is exceeded,
	// ErrExecutionLimitExceeded is returned.
	MaxOps int

	// MaxMemory can be set to a positive value to limit the total number of
	// bytes allocated by object-creating operators.  If this limit is
	// exceeded, a limitcheck error is returned.
	MaxMemory int64

	// budget tracks memory allocated by object-creating operators.  It is
	// created lazily from MaxMemory on the first call to Execute and is nil
	// when MaxMemory is not positive (unlimited).
	budget *membudget.Budget

	// Stack is the PostScript operand stack.
	Stack []Object

	// DictStack is the PostScript dictionary stack.
	DictStack []Dict

	// NumOps is the number of executed operations so far, counted as for
	// MaxOps.
	NumOps int

	// SystemDict is the PostScript system dictionary.
	SystemDict Dict

	// InternalDict is the PostScript internal dictionary.
	InternalDict Dict

	// UserDict is the PostScript user dictionary.
	UserDict Dict

	// ErrorDict is the PostScript error dictionary.
	ErrorDict Dict

	// Resources is a Dict of Dicts, which contains the resources
	// for each category.
	Resources Dict

	// FontDirectory is the PostScript font directory.
	// The `definefont` PostScript operator adds fonts to this dictionary.
	FontDirectory Dict

	// CMapDirectory is the PostScript CMap directory. This maps CMap names to
	// CMap dictionaries for all CMaps currently known to the interpreter.
	//
	// The "defineresource" PostScript operator can be used to add CMaps to
	// this dictionary.
	//
	// The keys in this dictionary are CMap names, and the values are CMap
	// dictionaries.  CMap dictionaries are documented in section 5.11.4 (CMap
	// Dictionaries) of the PostScript Language Reference Manual.  The
	// "CodeMap" field of the CMAP dictionaries can be cast to a '*CMapInfo'
	// object, which contains the mapping data.
	CMapDirectory Dict

	// DSC contains all DSC comments found in the input so far.
	// These are comments of the form "%%key: value" or "%%key".
	DSC []Comment

	errors []*postScriptError
	files  []*file // the files being executed, innermost last

	execStackDepth int

	// maxStack is the operand stack limit.
	// Zero means maxOperandStackDepth.
	maxStack int

	packing bool // the array packing mode, which has no effect
	global  bool // the VM allocation mode, which has no effect

	// These variables hold temporary data while a CMap is being defined.
	cmapMappings *CMapInfo    // the CMap being built, or nil
	cmapDict     Dict         // the dictionary which receives the CMap
	cmapFont     *CMapEntries // the font selected by usefont, nil if invalid
	cmapBlock    *cmapBlock   // the open begin…end block, or nil
	cmapMark     int          // the operand stack height at the start of cmapBlock
	cmapDefined  []Name       // the CMaps added by defineresource, in order
}

// NewInterpreter creates a new instance of the PostScript interpreter.
func NewInterpreter() *Interpreter {
	systemDict := makeSystemDict()
	userDict := systemDict["userdict"].(Dict)

	fontDirectory := systemDict["FontDirectory"].(Dict)
	cmapDirectory := Dict{}
	resources := Dict{
		"Font":    fontDirectory,
		"CIDFont": Dict{},
		"CMap":    cmapDirectory,
		"ProcSet": Dict{
			"CIDInit": maps.Clone(cidInit),
		},
	}

	intp := &Interpreter{
		DictStack: []Dict{
			systemDict,
			userDict,
		},
		FontDirectory: fontDirectory,
		CMapDirectory: cmapDirectory,
		Resources:     resources,
		SystemDict:    systemDict,
		InternalDict:  Dict{},
		UserDict:      userDict,
		ErrorDict:     systemDict["errordict"].(Dict),
	}

	for _, name := range allErrors {
		intp.ErrorDict[name] = defaultErrorHandler
	}

	return intp
}

// ExecuteString executes the PostScript code in the given string.
func (intp *Interpreter) ExecuteString(code string) error {
	return intp.Execute(strings.NewReader(code))
}

// Execute executes the PostScript code in the given reader.
func (intp *Interpreter) Execute(r io.Reader) error {
	if intp.MaxMemory > 0 && intp.budget == nil {
		intp.budget = membudget.New(intp.MaxMemory)
	}

	s := newScanner(r)
	err := intp.executeFile(&file{s: s})
	switch err {
	case errExit:
		err = intp.e(eInvalidexit, "exit outside loop")
	case errStop:
		err = nil
	}

	// the comments seen so far are kept even if execution fails
	intp.DSC = append(intp.DSC, s.DSC...)

	return err
}

func (intp *Interpreter) executeFile(f *file) error {
	s := f.s
	if intp.CheckStart {
		head := s.PeekN(2)
		if string(head) != "%!" {
			err := s.err
			if err == nil || err == io.EOF {
				err = ErrNoPostScript
			}
			return err
		}
		intp.CheckStart = false
	}

	intp.files = append(intp.files, f)
	defer func() {
		intp.files = intp.files[:len(intp.files)-1]
		f.closed = true
	}()

	// Procedures are built while reading the file, not by the executor:
	// braces only have their special meaning when read from a file.  The
	// procedures still being read are kept in open, innermost last.  They
	// stay off the operand stack, so that an error leaves the stack
	// unchanged.
	var open []Procedure
	for !f.isClosed() {
		o, err := s.ScanToken()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		switch o {
		case Operator("{"):
			open = append(open, Procedure{})
			continue
		case Operator("}"):
			if len(open) == 0 {
				return intp.e(eSyntaxerror, "unmatched '}'")
			}
			o = open[len(open)-1]
			open = open[:len(open)-1]
		}

		if len(open) == 0 {
			err = intp.executeOne(o, false)
		} else {
			err = intp.appendToken(&open[len(open)-1], o)
		}
		if err != nil {
			return err
		}
	}
	if len(open) > 0 {
		return intp.e(eSyntaxerror, "unterminated procedure")
	}
	return nil
}

func (intp *Interpreter) executeOne(obj Object, execProc bool) error {
	if execProc {
		if intp.execStackDepth >= 100 {
			return intp.e(eExecstackoverflow, "exec stack overflow")
		}
		intp.execStackDepth++
		defer func() { intp.execStackDepth-- }()
	}

	if err := intp.checkStack(); err != nil {
		return err
	}

recurseTail:
	intp.NumOps++
	if intp.MaxOps > 0 && intp.NumOps > intp.MaxOps {
		return ErrExecutionLimitExceeded
	}

	switch o := obj.(type) {
	case Operator:
		val, err := intp.loadName(Name(o))
		if err != nil {
			return err
		}
		obj = val
		execProc = true
		goto recurseTail

	case *builtin:
		err := o.fn(intp)
		if e2, ok := err.(*postScriptError); ok {
			level := len(intp.errors)
			if level < 5 {
				intp.errors = append(intp.errors, e2)
				if proc, ok := intp.ErrorDict[e2.tp]; ok {
					err = intp.executeOne(proc, true)
				}
				intp.errors = intp.errors[:level]
			}
		}
		return err

	case Procedure:
		if execProc {
			if len(o) == 0 {
				return nil
			}

			// use tail recursion
			for _, token := range o[:len(o)-1] {
				err := intp.executeOne(token, false)
				if err != nil {
					return err
				}
			}
			// the last element runs like the others: a procedure is pushed
			obj = o[len(o)-1]
			execProc = false
			goto recurseTail
		} else {
			intp.Stack = append(intp.Stack, o)
		}

	default:
		intp.Stack = append(intp.Stack, o)
	}
	return nil
}

// appendToken adds o to a procedure which is being read from a file.
func (intp *Interpreter) appendToken(proc *Procedure, o Object) error {
	if len(*proc) >= maxArraySize {
		return intp.e(eLimitcheck, "procedure too long")
	}
	if err := intp.charge(objectSize); err != nil {
		return err
	}
	*proc = append(*proc, o)
	return nil
}

// checkStack reports a stackoverflow error if the operand stack holds more
// than the permitted number of entries.  Pushes are not checked one by one:
// an operator may push past the limit, and the overflow is reported when a
// later object executes.
func (intp *Interpreter) checkStack() error {
	if len(intp.Stack) > intp.stackLimit() {
		return intp.e(eStackoverflow, "operand stack overflow")
	}
	return nil
}

// stackLimit returns the largest allowed operand stack depth.
func (intp *Interpreter) stackLimit() int {
	if intp.maxStack == 0 {
		return maxOperandStackDepth
	}
	return intp.maxStack
}

// loadName looks up name in the dictionary stack, topmost dictionary first.
// The key is taken as a name rather than an [Object] so that callers, which
// always know the concrete type, need not box it into an interface value.
func (intp *Interpreter) loadName(name Name) (Object, error) {
	for _, d := range slices.Backward(intp.DictStack) {
		if val, ok := d[name]; ok {
			return val, nil
		}
	}
	return nil, intp.e(eUndefined, "load: %s not defined", name)
}

const (
	maxArraySize         = 65536
	maxDictSize          = 65536
	maxDictStackDepth    = 20
	maxOperandStackDepth = 500
	maxStringSize        = 65536
)

// approximate per-element weights for the memory budget; not exact
// unsafe.Sizeof values, just accounting weights that bound amplification
const (
	objectSize    = 16                         // an Object interface value
	dictEntrySize = membudget.MapEntryOverhead // a dictionary entry
)

// charge accounts for n bytes against the memory budget.  It returns a
// limitcheck error if the budget is exceeded, and nil when no budget is set.
func (intp *Interpreter) charge(n int) error {
	if intp.budget == nil {
		return nil
	}
	if err := intp.budget.Charge(n); err != nil {
		return intp.e(eLimitcheck, "memory limit exceeded")
	}
	return nil
}

// workPerStep is the number of elements or bytes an operator can touch for
// the cost of one step of the operation budget.
const workPerStep = 64

// chargeWork accounts for an operator which touches n elements or bytes.
// It adds n/workPerStep steps, rounded down, to the step the operator
// itself already counts.  It returns ErrExecutionLimitExceeded if this
// exceeds MaxOps.
func (intp *Interpreter) chargeWork(n int) error {
	if n <= 0 {
		return nil
	}
	intp.NumOps += n / workPerStep
	if intp.MaxOps > 0 && intp.NumOps > intp.MaxOps {
		return ErrExecutionLimitExceeded
	}
	return nil
}
