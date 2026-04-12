// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"sort"
	"testing"
)

// sortedInts returns a copy of in sorted ascending so merge output (which
// uses map iteration and has no guaranteed order) can be compared against
// an expected slice deterministically.
func sortedInts(in []int) []int {
	out := append([]int(nil), in...)
	sort.Ints(out)
	return out
}

func sortedStrings(in [][]byte) []string {
	out := make([]string, len(in))
	for i, b := range in {
		out[i] = string(b)
	}
	sort.Strings(out)
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLookupDelta_EmptyDeltaReturnsCommitted(t *testing.T) {
	d := newLookupDelta[int, string]()
	committed := []int{1, 2, 3}
	got := d.merge(committed, []string{"a"})
	if !equalInts(sortedInts(got), sortedInts(committed)) {
		t.Fatalf("empty delta should pass through committed; got %v, want %v", got, committed)
	}
}

func TestLookupDelta_StagedInsertAppearsInResult(t *testing.T) {
	d := newLookupDelta[int, string]()
	d.stageSet(10, "a")
	got := d.merge([]int{1, 2}, []string{"a"})
	want := []int{1, 2, 10}
	if !equalInts(sortedInts(got), want) {
		t.Fatalf("insert: got %v, want %v", sortedInts(got), want)
	}
}

func TestLookupDelta_StagedInsertIsolatedByValue(t *testing.T) {
	d := newLookupDelta[int, string]()
	d.stageSet(10, "a")
	got := d.merge([]int{1, 2}, []string{"b"})
	want := []int{1, 2}
	if !equalInts(sortedInts(got), want) {
		t.Fatalf("insert under other value should not leak into 'b' query; got %v, want %v", sortedInts(got), want)
	}
}

func TestLookupDelta_StagedDeleteRemovesCommitted(t *testing.T) {
	d := newLookupDelta[int, string]()
	d.stageDelete(2)
	got := d.merge([]int{1, 2, 3}, []string{"a"})
	want := []int{1, 3}
	if !equalInts(sortedInts(got), want) {
		t.Fatalf("delete should remove from committed; got %v, want %v", sortedInts(got), want)
	}
}

func TestLookupDelta_UpdateMovesKeyBetweenBuckets(t *testing.T) {
	d := newLookupDelta[int, string]()
	d.stageSet(1, "b")

	gotOld := d.merge([]int{1, 2}, []string{"a"})
	wantOld := []int{2}
	if !equalInts(sortedInts(gotOld), wantOld) {
		t.Fatalf("old-value query should exclude updated key; got %v, want %v", sortedInts(gotOld), wantOld)
	}

	gotNew := d.merge([]int{}, []string{"b"})
	wantNew := []int{1}
	if !equalInts(sortedInts(gotNew), wantNew) {
		t.Fatalf("new-value query should include updated key; got %v, want %v", sortedInts(gotNew), wantNew)
	}
}

func TestLookupDelta_SetThenDeleteErasesKey(t *testing.T) {
	d := newLookupDelta[int, string]()
	d.stageSet(5, "a")
	d.stageDelete(5)
	got := d.merge([]int{}, []string{"a"})
	if len(got) != 0 {
		t.Fatalf("set-then-delete should leave no trace; got %v", got)
	}
	if _, ok := d.forward["a"]; ok {
		t.Fatalf("forward bucket 'a' should be empty after delete")
	}
}

func TestLookupDelta_DoubleSetOverwrites(t *testing.T) {
	d := newLookupDelta[int, string]()
	d.stageSet(5, "a")
	d.stageSet(5, "b")

	gotA := d.merge([]int{}, []string{"a"})
	if len(gotA) != 0 {
		t.Fatalf("double-set: 'a' query should see nothing; got %v", gotA)
	}
	gotB := d.merge([]int{}, []string{"b"})
	want := []int{5}
	if !equalInts(gotB, want) {
		t.Fatalf("double-set: 'b' query should see key; got %v, want %v", gotB, want)
	}
	if _, ok := d.forward["a"]; ok {
		t.Fatalf("forward['a'] should be empty after overwrite")
	}
}

func TestLookupDelta_MultiValueQueryUnionsStaged(t *testing.T) {
	d := newLookupDelta[int, string]()
	d.stageSet(10, "a")
	d.stageSet(20, "b")
	d.stageSet(30, "c")
	got := d.merge([]int{1}, []string{"a", "b"})
	want := []int{1, 10, 20}
	if !equalInts(sortedInts(got), want) {
		t.Fatalf("multi-value query: got %v, want %v", sortedInts(got), want)
	}
}

func TestLookupDelta_CommittedThenDeleteUsesEntryValueCheck(t *testing.T) {
	d := newLookupDelta[int, string]()
	d.stageDelete(1)
	got := d.merge([]int{1}, []string{"a"})
	if len(got) != 0 {
		t.Fatalf("committed key staged for delete should be excluded; got %v", got)
	}
}

func TestBytesLookupDelta_EmptyDeltaReturnsCommitted(t *testing.T) {
	d := newBytesLookupDelta[string]()
	committed := [][]byte{[]byte("k1"), []byte("k2")}
	got := d.merge(committed, []string{"a"})
	want := []string{"k1", "k2"}
	if !equalStrings(sortedStrings(got), want) {
		t.Fatalf("empty delta: got %v, want %v", sortedStrings(got), want)
	}
}

func TestBytesLookupDelta_StagedInsertAppears(t *testing.T) {
	d := newBytesLookupDelta[string]()
	d.stageSet([]byte("new"), "a")
	got := d.merge([][]byte{[]byte("k1")}, []string{"a"})
	want := []string{"k1", "new"}
	if !equalStrings(sortedStrings(got), want) {
		t.Fatalf("insert: got %v, want %v", sortedStrings(got), want)
	}
}

func TestBytesLookupDelta_StagedDeleteRemovesCommitted(t *testing.T) {
	d := newBytesLookupDelta[string]()
	d.stageDelete([]byte("k2"))
	got := d.merge([][]byte{[]byte("k1"), []byte("k2"), []byte("k3")}, []string{"a"})
	want := []string{"k1", "k3"}
	if !equalStrings(sortedStrings(got), want) {
		t.Fatalf("delete: got %v, want %v", sortedStrings(got), want)
	}
}

func TestBytesLookupDelta_UpdateMovesBetweenBuckets(t *testing.T) {
	d := newBytesLookupDelta[string]()
	d.stageSet([]byte("k1"), "b")

	gotOld := d.merge([][]byte{[]byte("k1"), []byte("k2")}, []string{"a"})
	wantOld := []string{"k2"}
	if !equalStrings(sortedStrings(gotOld), wantOld) {
		t.Fatalf("old-value: got %v, want %v", sortedStrings(gotOld), wantOld)
	}

	gotNew := d.merge(nil, []string{"b"})
	wantNew := []string{"k1"}
	if !equalStrings(sortedStrings(gotNew), wantNew) {
		t.Fatalf("new-value: got %v, want %v", sortedStrings(gotNew), wantNew)
	}
}

func TestBytesLookupDelta_InsertKeyIsCopied(t *testing.T) {
	d := newBytesLookupDelta[string]()
	scratch := []byte("mut")
	d.stageSet(scratch, "a")
	scratch[0] = 'X'

	got := d.merge(nil, []string{"a"})
	if len(got) != 1 || string(got[0]) != "mut" {
		t.Fatalf("staged key should be isolated from caller mutation; got %q", got)
	}
}

func TestBytesLookupDelta_ResultKeysAreFreshCopies(t *testing.T) {
	d := newBytesLookupDelta[string]()
	d.stageSet([]byte("a1"), "a")
	got := d.merge(nil, []string{"a"})
	if len(got) != 1 {
		t.Fatalf("expected 1 key, got %v", got)
	}
	got[0][0] = 'Z'

	again := d.merge(nil, []string{"a"})
	if len(again) != 1 || string(again[0]) != "a1" {
		t.Fatalf("second merge should return a fresh copy unaffected by caller mutation; got %q", again)
	}
}
