// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package meta_test

import (
	jsonv2 "encoding/json/v2"
	"reflect"
	"testing"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

// FuzzRead checks that Read takes any meta file, and that a channel it reads and that
// validates writes and reads back equal.
func FuzzRead(f *testing.F) {
	codec := json.NewCodec(jsonv2.MatchCaseInsensitiveNames(true))
	for _, ch := range []channel.Channel{
		{Key: 1, Name: "clock", DataType: telem.TimestampT, IsIndex: true},
		{Key: 2, Name: "data", DataType: telem.Float64T, Index: 1},
		{Key: 3, Name: "virtual", DataType: telem.StringT, Virtual: true},
	} {
		data, err := codec.Encode(ch)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		in := fs.NewMem()
		if err := fs.WriteFile(in, meta.File, data); err != nil {
			t.Fatal(err)
		}
		ch, err := meta.Read(t.Context(), in, codec)
		if err != nil || ch.Validate() != nil {
			return
		}
		out := fs.NewMem()
		if err = meta.Create(t.Context(), out, codec, ch); err != nil {
			t.Fatal(err)
		}
		back, err := meta.Read(t.Context(), out, codec)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(back, ch) {
			t.Fatalf("read back %+v, want %+v", back, ch)
		}
	})
}
