// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain_test

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"

	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

// twoDomains is the index file of two domains, 1 s to 2 s and 2 s to 3 s, of 3 bytes
// each in file 1.
var twoDomains = []byte{
	0x00, 0xca, 0x9a, 0x3b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x94, 0x35, 0x77, 0x00,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
	0x00, 0x94, 0x35, 0x77, 0x00, 0x00, 0x00, 0x00, 0x00, 0x5e, 0xd0, 0xb2, 0x00,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x03, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
}

// ranges returns the time range of each domain in db, in order, and false when the
// iterator takes more than limit steps. An index that Open loads with no check can
// make Next return the same domain forever.
func ranges(t *testing.T, db *domain.DB, limit int) ([]telem.TimeRange, bool) {
	i := db.OpenIterator(domain.IterRange(telem.TimeRangeMax))
	var out []telem.TimeRange
	for i.SeekFirst(t.Context()); i.Valid() && len(out) <= limit; i.Next() {
		out = append(out, i.TimeRange())
	}
	if err := i.Close(); err != nil {
		t.Fatal(err)
	}
	return out, len(out) <= limit
}

// apart reports whether each whole pointer of index spans a time range that is not
// empty and starts at or after the end of the one before it.
func apart(index []byte) bool {
	var last telem.TimeStamp
	for off := 0; off+26 <= len(index); off += 26 {
		start := telem.TimeStamp(binary.LittleEndian.Uint64(index[off:]))
		end := telem.TimeStamp(binary.LittleEndian.Uint64(index[off+8:]))
		if start >= end || (off > 0 && start < last) {
			return false
		}
		last = end
	}
	return true
}

// FuzzOpen checks that Open takes any index file without a panic. When its pointers
// are in order and apart, a domain written after them must read back after a reopen,
// with the loaded domains before it.
func FuzzOpen(f *testing.F) {
	f.Add([]byte{})
	f.Add(twoDomains)
	f.Add(twoDomains[:36])
	f.Fuzz(func(t *testing.T, index []byte) {
		fs := xfs.NewMem()
		if err := xfs.WriteFile(fs, "index.domain", index); err != nil {
			t.Fatal(err)
		}
		cfg := domain.Config{FS: fs, FileSize: telem.Megabyte}
		db, err := domain.Open(cfg)
		if err != nil {
			return
		}
		limit := len(index) / 26
		loaded, ended := ranges(t, db, limit)
		var end telem.TimeStamp
		if len(loaded) > 0 {
			end = loaded[len(loaded)-1].End
		}
		written := (end + telem.SecondTS).Range(end + 2*telem.SecondTS)
		if !ended || !apart(index) || end > telem.TimeStampMax-2*telem.SecondTS ||
			testutil.WriteDomain(t.Context(), db, written, []byte{1, 2, 3}) != nil {
			if err = db.Close(); err != nil {
				t.Fatal(err)
			}
			return
		}
		if err = db.Close(); err != nil {
			t.Fatal(err)
		}
		if db, err = domain.Open(cfg); err != nil {
			t.Fatalf("reopen: %v", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
		}()
		got, _ := ranges(t, db, limit+1)
		if want := append(loaded, written); !slices.Equal(got, want) {
			t.Fatalf("reopen loaded %v, want %v", got, want)
		}
		data, err := testutil.ReadDomain(t.Context(), db, written)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, []byte{1, 2, 3}) {
			t.Fatalf("read back %v, want [1 2 3]", data)
		}
	})
}
