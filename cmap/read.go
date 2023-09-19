package cmap

import (
	"fmt"
	"io"

	"seehuhn.de/go/postscript"
)

// Read reads the raw PostScript data of a CMap from an [io.Reader].
func Read(r io.Reader) (postscript.Dict, error) {
	intp := postscript.NewInterpreter()
	intp.MaxOps = 1_000_000 // TODO(voss): measure what is required
	err := intp.Execute(r)
	if err != nil {
		return nil, err
	}

	var cmap postscript.Dict
	for name, val := range intp.CMapDirectory {
		var ok bool
		cmap, ok = val.(postscript.Dict)
		if !ok {
			continue
		}
		if _, ok := cmap["CMapName"].(postscript.Name); !ok {
			cmap["CMapName"] = postscript.Name(name)
		}
	}
	if cmap == nil {
		return nil, fmt.Errorf("no valid CMap found")
	}

	return cmap, nil
}
