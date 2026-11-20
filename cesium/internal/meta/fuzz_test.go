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
	"bytes"
	jsonv2 "encoding/json/v2"
	"testing"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
)

// FuzzOpen checks that Open takes any meta file. A channel it opens validates, is at
// the current version, and opens again equal after Create. A channel that validates as
// it is stored opens unchanged but for its version. Open leaves the file as it was.
func FuzzOpen(f *testing.F) {
	for _, file := range []string{
		// Version 3, as Create writes it today.
		`{"name":"clock","data_type":"timestamp","key":1,"index":0,"is_index":true,` +
			`"virtual":false,"concurrency":0,"version":3}`,
		// Version 2, stored under Go field names.
		`{"name":"Faraday","data_type":"timestamp","key":2,"index":0,"IsIndex":true,` +
			`"version":2}`,
		// Version 2, a virtual channel that also claims to be an index.
		`{"name":"Faraday","data_type":"timestamp","key":3,"index":0,"IsIndex":true,` +
			`"Virtual":true,"version":2}`,
		// Version 1, stored under Go field names.
		`{"name":"Faraday","data_type":"int64","key":4,"index":0,"IsIndex":true,` +
			`"Virtual":false,"Concurrency":1,"version":1}`,
		// Version 0, with no name.
		`{"data_type":"float64","key":5,"index":1}`,
	} {
		f.Add([]byte(file))
	}
	codec := json.NewCodec(jsonv2.MatchCaseInsensitiveNames(true))
	f.Fuzz(func(t *testing.T, data []byte) {
		in := fs.NewMem()
		if err := fs.WriteFile(in, meta.File, data); err != nil {
			t.Fatal(err)
		}
		ch, err := meta.Open(t.Context(), in, channel.Channel{}, codec)
		if errors.Is(err, meta.ErrIgnoreChannel) {
			return
		}
		stored, readErr := meta.Read(t.Context(), in, codec)
		if readErr == nil {
			stored.Version = channel.VersionCurrent
			if stored.Validate() == nil && (err != nil || ch != stored) {
				t.Fatalf("open gave %+v (%v), want the stored %+v", ch, err, stored)
			}
		}
		if err != nil {
			return
		}
		if err = ch.Validate(); err != nil {
			t.Fatalf("open gave a channel that does not validate: %v", err)
		}
		if ch.Version != channel.VersionCurrent {
			t.Fatalf(
				"open gave version %d, want %d",
				ch.Version,
				channel.VersionCurrent,
			)
		}
		if after, err := fs.ReadFile(
			in,
			meta.File,
		); err != nil ||
			!bytes.Equal(after, data) {
			t.Fatalf("open changed the file to %q (%v)", after, err)
		}
		out := fs.NewMem()
		if err = meta.Create(t.Context(), out, codec, ch); err != nil {
			t.Fatal(err)
		}
		back, err := meta.Open(t.Context(), out, channel.Channel{}, codec)
		if err != nil || back != ch {
			t.Fatalf("open after create gave %+v (%v), want %+v", back, err, ch)
		}
	})
}
