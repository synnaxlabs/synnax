// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package orc_test

import (
	"context"
	"testing"

	"github.com/synnaxlabs/x/encoding/orc"
)

// benchRecord has a field mix representative of a gorp entry: fixed-size scalars plus
// variable-length strings.
type benchRecord struct {
	ID      uint64
	Name    string
	Comment string
	Value   float64
	Active  bool
}

func (r *benchRecord) EncodeOrc(w *orc.Writer) error {
	w.Uint64(r.ID)
	w.String(r.Name)
	w.String(r.Comment)
	w.Float64(r.Value)
	w.Bool(r.Active)
	return nil
}

func (r *benchRecord) DecodeOrc(rd *orc.Reader) error {
	var err error
	if r.ID, err = rd.Uint64(); err != nil {
		return err
	}
	if r.Name, err = rd.String(); err != nil {
		return err
	}
	if r.Comment, err = rd.String(); err != nil {
		return err
	}
	if r.Value, err = rd.Float64(); err != nil {
		return err
	}
	r.Active, err = rd.Bool()
	return err
}

func benchInput() *benchRecord {
	return &benchRecord{
		ID:      42,
		Name:    "bench-record",
		Comment: "a moderately sized comment string for realistic payloads",
		Value:   3.14159,
		Active:  true,
	}
}

func BenchmarkCodecEncode(b *testing.B) {
	ctx := context.Background()
	in := benchInput()
	if _, err := orc.Codec.Encode(ctx, in); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := orc.Codec.Encode(ctx, in); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCodecDecodeMiss measures the magic-header rejection path, which the decode
// fallback in gorp hits for every row written by another codec.
func BenchmarkCodecDecodeMiss(b *testing.B) {
	ctx := context.Background()
	data := []byte{0x00, 0x01, 0x02, 0x03}
	out := &benchRecord{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := orc.Codec.Decode(ctx, data, out); err == nil {
			b.Fatal("expected magic mismatch error")
		}
	}
}

func BenchmarkCodecDecode(b *testing.B) {
	ctx := context.Background()
	data, err := orc.Codec.Encode(ctx, benchInput())
	if err != nil {
		b.Fatal(err)
	}
	out := &benchRecord{}
	if err := orc.Codec.Decode(ctx, data, out); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := orc.Codec.Decode(ctx, data, out); err != nil {
			b.Fatal(err)
		}
	}
}
