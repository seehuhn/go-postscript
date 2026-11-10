// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
	"errors"
	"fmt"
)

var defaultErrorHandler = builtin(defaultErrorHandlerFn)

func defaultErrorHandlerFn(intp *Interpreter) error {
	return intp.errors[len(intp.errors)-1]
}

func (intp *Interpreter) e(tp Name, format string, a ...any) error {
	return &postScriptError{tp, fmt.Sprintf(format, a...)}
}

type postScriptError struct {
	tp  Name
	msg string
}

func (err *postScriptError) Error() string {
	return fmt.Sprintf("%s: %s", string(err.tp), err.msg)
}

var (
	eConfigurationerror = Name("configurationerror")
	eDictfull           = Name("dictfull")
	eDictstackoverflow  = Name("dictstackoverflow")
	eDictstackunderflow = Name("dictstackunderflow")
	eExecstackoverflow  = Name("execstackoverflow")
	eHandleerror        = Name("handleerror")
	eInterrupt          = Name("interrupt")
	eInvalidaccess      = Name("invalidaccess")
	eInvalidexit        = Name("invalidexit")
	eInvalidfileaccess  = Name("invalidfileaccess")
	eInvalidfont        = Name("invalidfont")
	eInvalidrestore     = Name("invalidrestore")
	eIoerror            = Name("ioerror")
	eLimitcheck         = Name("limitcheck")
	eNocurrentpoint     = Name("nocurrentpoint")
	eRangecheck         = Name("rangecheck")
	eStackoverflow      = Name("stackoverflow")
	eStackunderflow     = Name("stackunderflow")
	eSyntaxerror        = Name("syntaxerror")
	eTimeout            = Name("timeout")
	eTypecheck          = Name("typecheck")
	eUndefined          = Name("undefined")
	eUndefinedfilename  = Name("undefinedfilename")
	eUndefinedresource  = Name("undefinedresource")
	eUndefinedresult    = Name("undefinedresult")
	eUnmatchedmark      = Name("unmatchedmark")
	eUnregistered       = Name("unregistered")
	eVMerror            = Name("VMerror")
)

var allErrors = []Name{
	eConfigurationerror,
	eDictfull,
	eDictstackoverflow,
	eDictstackunderflow,
	eExecstackoverflow,
	eHandleerror,
	eInterrupt,
	eInvalidaccess,
	eInvalidexit,
	eInvalidfileaccess,
	eInvalidfont,
	eInvalidrestore,
	eIoerror,
	eLimitcheck,
	eNocurrentpoint,
	eRangecheck,
	eStackoverflow,
	eStackunderflow,
	eSyntaxerror,
	eTimeout,
	eTypecheck,
	eUndefined,
	eUndefinedfilename,
	eUndefinedresource,
	eUndefinedresult,
	eUnmatchedmark,
	eUnregistered,
	eVMerror,
}

var (
	ErrExecutionLimitExceeded = &postScriptError{eInterrupt, "execution limit exceeded"}
	ErrNoPostScript           = errors.New("not a PostScript file")
)
