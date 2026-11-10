// Copyright 2023 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"seehuhn.de/go/postscript"
	"seehuhn.de/go/postscript/pfb"
)

func main() {
	for _, fname := range os.Args[1:] {
		err := showInfo(fname)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func showInfo(fname string) error {
	fd, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer fd.Close()

	// read the first byte of fd to determine the file type
	var buf [1]byte
	_, err = io.ReadFull(fd, buf[:])
	if err != nil {
		return err
	}
	fd.Seek(0, 0)

	var r io.Reader = fd
	if buf[0] == 0x80 {
		r = pfb.Decode(r)
	}

	intp := postscript.NewInterpreter()
	err = intp.Execute(r)
	if err != nil {
		return err
	}

	for key, font := range intp.FontDirectory {
		fmt.Printf("# %s\n", key)
		fontDict := font.(postscript.Dict)
		for key, val := range fontDict {
			if key == "Private" || key == "FontInfo" {
				fmt.Println(string(key) + ":")
				for k2, v2 := range val.(postscript.Dict) {
					valString := fmt.Sprint(v2)
					if len(valString) > 70 {
						valString = fmt.Sprintf("<%T>", val)
					}
					fmt.Println("  "+string(k2)+":", valString)
				}
				continue
			}
			valString := fmt.Sprint(val)
			if len(valString) > 70 {
				valString = fmt.Sprintf("<%T>", val)
			}
			fmt.Println(string(key)+":", valString)
		}
		fmt.Println()
	}
	return nil
}
