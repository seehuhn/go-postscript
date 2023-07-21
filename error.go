// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2023  Jochen Voss <voss@seehuhn.de>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package postscript

import "fmt"

func defaultErrorHandler(intp *Interpreter) error {
	return intp.errors[len(intp.errors)-1]
}

func (intp *Interpreter) E(tp Name, format string, a ...any) error {
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
