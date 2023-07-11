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

type postScriptError struct {
	tp Name
}

func (err *postScriptError) Error() string {
	return string(err.tp)
}

var (
	errConfigurationerror = &postScriptError{Name("configurationerror")}
	errDictfull           = &postScriptError{Name("dictfull")}
	errDictstackoverflow  = &postScriptError{Name("dictstackoverflow")}
	errDictstackunderflow = &postScriptError{Name("dictstackunderflow")}
	errExecstackoverflow  = &postScriptError{Name("execstackoverflow")}
	errHandleerror        = &postScriptError{Name("handleerror")}
	errInterrupt          = &postScriptError{Name("interrupt")}
	errInvalidaccess      = &postScriptError{Name("invalidaccess")}
	errInvalidexit        = &postScriptError{Name("invalidexit")}
	errInvalidfileaccess  = &postScriptError{Name("invalidfileaccess")}
	errInvalidfont        = &postScriptError{Name("invalidfont")}
	errInvalidrestore     = &postScriptError{Name("invalidrestore")}
	errIoerror            = &postScriptError{Name("ioerror")}
	errLimitcheck         = &postScriptError{Name("limitcheck")}
	errNocurrentpoint     = &postScriptError{Name("nocurrentpoint")}
	errRangecheck         = &postScriptError{Name("rangecheck")}
	errStackoverflow      = &postScriptError{Name("stackoverflow")}
	errStackunderflow     = &postScriptError{Name("stackunderflow")}
	errSyntaxerror        = &postScriptError{Name("syntaxerror")}
	errTimeout            = &postScriptError{Name("timeout")}
	errTypecheck          = &postScriptError{Name("typecheck")}
	errUndefined          = &postScriptError{Name("undefined")}
	errUndefinedfilename  = &postScriptError{Name("undefinedfilename")}
	errUndefinedresource  = &postScriptError{Name("undefinedresource")}
	errUndefinedresult    = &postScriptError{Name("undefinedresult")}
	errUnmatchedmark      = &postScriptError{Name("unmatchedmark")}
	errUnregistered       = &postScriptError{Name("unregistered")}
	errVMerror            = &postScriptError{Name("VMerror")}
)

var allErrors = []*postScriptError{
	errConfigurationerror,
	errDictfull,
	errDictstackoverflow,
	errDictstackunderflow,
	errExecstackoverflow,
	errHandleerror,
	errInterrupt,
	errInvalidaccess,
	errInvalidexit,
	errInvalidfileaccess,
	errInvalidfont,
	errInvalidrestore,
	errIoerror,
	errLimitcheck,
	errNocurrentpoint,
	errRangecheck,
	errStackoverflow,
	errStackunderflow,
	errSyntaxerror,
	errTimeout,
	errTypecheck,
	errUndefined,
	errUndefinedfilename,
	errUndefinedresource,
	errUndefinedresult,
	errUnmatchedmark,
	errUnregistered,
	errVMerror,
}
