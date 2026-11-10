// Copyright 2026 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package postscript

import (
	"strings"
	"testing"
)

// allocationBomb retains 65536 inner arrays of 65536 entries each.  Without a
// memory budget it allocates ~64 GiB (65536*65536*16 bytes) while staying well
// under the operation cap, OOM-killing the process.
const allocationBomb = "/big 65536 array def\n0 1 65535 { big exch 65536 array put } for\n"

// TestMaxMemoryLimit checks that a positive MaxMemory aborts the allocation
// bomb with an error instead of exhausting memory.  With the budget removed
// this test would OOM the process, which is the intended regression signal.
func TestMaxMemoryLimit(t *testing.T) {
	intp := NewInterpreter()
	intp.MaxOps = 1_000_000
	intp.MaxMemory = 8 << 20 // 8 MiB, far below the bomb's ~64 GiB
	if err := intp.ExecuteString(allocationBomb); err == nil {
		t.Error("expected memory limit error, got nil")
	}
}

// TestMaxMemoryUnlimited checks that the default (MaxMemory == 0) imposes no
// limit and ordinary programs run unaffected.
func TestMaxMemoryUnlimited(t *testing.T) {
	intp := NewInterpreter()
	if err := intp.ExecuteString("/a 1000 array def /s 1000 string def"); err != nil {
		t.Fatal(err)
	}
}

// TestMaxMemoryAllowsModestUse checks that a legitimate program comfortably
// fits within a modest budget, so the accounting does not reject valid input.
func TestMaxMemoryAllowsModestUse(t *testing.T) {
	intp := NewInterpreter()
	intp.MaxMemory = 1 << 20 // 1 MiB
	// a few small composites, nowhere near the budget
	if err := intp.ExecuteString("/a 100 array def /d 50 dict def << /K (v) >> pop"); err != nil {
		t.Fatal(err)
	}
}

// TestReadCMapMemoryLimit checks that the bomb fails through the ReadCMap entry
// point (the path reached from untrusted PDF fonts) rather than exhausting
// memory.
func TestReadCMapMemoryLimit(t *testing.T) {
	_, _, err := ReadCMap(strings.NewReader(allocationBomb))
	if err == nil {
		t.Error("expected error from ReadCMap, got nil")
	}
}
