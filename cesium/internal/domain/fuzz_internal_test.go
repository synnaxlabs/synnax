// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import (
	"bytes"
	"testing"
)

// FuzzPointerCodec checks that the decode of any bytes encodes back to its whole
// pointers.
func FuzzPointerCodec(f *testing.F) {
	var codec pointerCodec
	f.Add([]byte{})
	f.Add(codec.encode(nil, 0, []pointer{
		{Start: 1, End: 5, fileKey: 1, size: 32},
		{Start: 5, End: 9, fileKey: 2, offset: 32, size: 8},
	}))
	f.Fuzz(func(t *testing.T, b []byte) {
		whole := b[:len(b)/pointerByteSize*pointerByteSize]
		if got := codec.encode(nil, 0, codec.decode(b)); !bytes.Equal(got, whole) {
			t.Fatalf("decode, then encode, gave %x, want %x", got, whole)
		}
	})
}
