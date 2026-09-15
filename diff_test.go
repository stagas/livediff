package main

import "testing"

func TestDiffReportsChangedLinesWithContext(t *testing.T) {
	result := makeHunks(diffLines("one\ntwo\nthree\nfour\n", "one\nTWO\nthree\nfour\n"), 1)
	if len(result) != 1 {
		t.Fatalf("got %d hunks, want 1", len(result))
	}
	want := []diffLine{
		{kind: equal, text: "one"},
		{kind: remove, text: "two"},
		{kind: add, text: "TWO"},
		{kind: equal, text: "three"},
	}
	if len(result[0].lines) != len(want) {
		t.Fatalf("got %d lines, want %d", len(result[0].lines), len(want))
	}
	for index := range want {
		if result[0].lines[index] != want[index] {
			t.Errorf("line %d: got %#v, want %#v", index, result[0].lines[index], want[index])
		}
	}
}

func TestDistantChangesProduceSeparateHunks(t *testing.T) {
	before := "0\n1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11"
	after := "0\nx\n2\n3\n4\n5\n6\n7\n8\n9\ny\n11"
	if result := makeHunks(diffLines(before, after), 1); len(result) != 2 {
		t.Fatalf("got %d hunks, want 2", len(result))
	}
}

func TestCompactHunksHaveNoContext(t *testing.T) {
	result := makeHunks(diffLines("one\ntwo\nthree\n", "one\nTWO\nthree\n"), 0)
	for _, line := range result[0].lines {
		if line.kind == equal {
			t.Fatal("zero-context hunk contains an unchanged line")
		}
	}
}
