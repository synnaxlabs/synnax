// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp

import (
	"math/rand/v2"
	"reflect"
	"testing"
	"unicode/utf8"
)

const quoteDiffSeed = "func f() {\n" +
	"    x := 'ox-pt-1' + 'pt-2'\n" +
	"    s := `first\n" +
	"second 'quoted' \"text\"\n" +
	"`\n" +
	"    /* a 'comment'\n" +
	"    still */ y := \"it's\" // don't\n" +
	"    z := f\"{'ox-p'} and {x}\"\n" +
	"    m := f`{'pt-2'}\n" +
	"line`\n" +
	"}\n"

var quoteDiffPieces = []string{
	"`", "'", "\"", "/*", "*/", "//", "\n", "x", "{", "}", "f\"{", "\\", "'ox-p'",
	"é", " ", "é", "`a`", "\"b\"", "\r\n",
}

// runeBoundary moves i back to the start of the rune that holds it.
func runeBoundary(text string, i int) int {
	for i > 0 && i < len(text) && !utf8.RuneStart(text[i]) {
		i--
	}
	return i
}

// TestQuoteScanUpdateMatchesFullScan checks that an update after each of many edits
// gives the facts of a full scan of the edited text.
func TestQuoteScanUpdateMatchesFullScan(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 13))
	scanner := newLineScanner()
	text := quoteDiffSeed
	scan := scanQuotes(text, scanner)
	for n := range 3000 {
		if len(text) > 2000 {
			text = quoteDiffSeed
			scan = scanQuotes(text, scanner)
		}
		at := runeBoundary(text, r.IntN(len(text)+1))
		end := runeBoundary(text, min(len(text), at+r.IntN(6)))
		if end < at {
			end = at
		}
		insert := ""
		for range r.IntN(3) {
			insert += quoteDiffPieces[r.IntN(len(quoteDiffPieces))]
		}
		next := text[:at] + insert + text[end:]
		updated := scan.update(text, next, scanner)
		fresh := scanQuotes(next, newLineScanner())
		if !reflect.DeepEqual(updated.lines, fresh.lines) {
			t.Fatalf(
				"edit %d: replace [%d, %d) of %q with %q: the update differs from a full scan",
				n,
				at,
				end,
				text,
				insert,
			)
		}
		text, scan = next, updated
	}
}
