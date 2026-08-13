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
	"bytes"
	"context"
	"testing"

	"github.com/synnaxlabs/x/encoding/orc"
)

// benchRecord exercises every Writer and Reader method: all fixed-size primitives,
// strings, raw and length-prefixed bytes, collections, and depth-guarded recursion.
type benchRecord struct {
	U8    uint8
	U16   uint16
	U32   uint32
	U64   uint64
	I8    int8
	I16   int16
	I32   int32
	I64   int64
	F32   float32
	F64   float64
	Flag  bool
	Name  string
	Blob  []byte
	Tags  []string
	Raw   [4]byte
	Child *benchRecord
}

func (r *benchRecord) EncodeOrc(w *orc.Writer) error {
	w.Resize(w.Len() + 128)
	w.Uint8(r.U8)
	w.Uint16(r.U16)
	w.Uint32(r.U32)
	w.Uint64(r.U64)
	w.Int8(r.I8)
	w.Int16(r.I16)
	w.Int32(r.I32)
	w.Int64(r.I64)
	w.Float32(r.F32)
	w.Float64(r.F64)
	w.Bool(r.Flag)
	w.String(r.Name)
	w.WriteWithLen(r.Blob)
	w.Uint32(uint32(len(r.Tags)))
	for _, t := range r.Tags {
		w.String(t)
	}
	w.Write(r.Raw[:])
	w.Bool(r.Child != nil)
	if r.Child != nil {
		return r.Child.EncodeOrc(w)
	}
	return nil
}

func (r *benchRecord) DecodeOrc(rd *orc.Reader) error {
	if err := rd.PushDepth(orc.MaxDecodeDepth); err != nil {
		return err
	}
	defer rd.PopDepth()
	var err error
	if r.U8, err = rd.Uint8(); err != nil {
		return err
	}
	if r.U16, err = rd.Uint16(); err != nil {
		return err
	}
	if r.U32, err = rd.Uint32(); err != nil {
		return err
	}
	if r.U64, err = rd.Uint64(); err != nil {
		return err
	}
	if r.I8, err = rd.Int8(); err != nil {
		return err
	}
	if r.I16, err = rd.Int16(); err != nil {
		return err
	}
	if r.I32, err = rd.Int32(); err != nil {
		return err
	}
	if r.I64, err = rd.Int64(); err != nil {
		return err
	}
	if r.F32, err = rd.Float32(); err != nil {
		return err
	}
	if r.F64, err = rd.Float64(); err != nil {
		return err
	}
	if r.Flag, err = rd.Bool(); err != nil {
		return err
	}
	if r.Name, err = rd.String(); err != nil {
		return err
	}
	if r.Blob, err = rd.ReadWithLen(); err != nil {
		return err
	}
	n, err := rd.CollectionLen()
	if err != nil {
		return err
	}
	r.Tags = make([]string, n)
	for i := range r.Tags {
		if r.Tags[i], err = rd.String(); err != nil {
			return err
		}
	}
	if _, err = rd.Read(r.Raw[:]); err != nil {
		return err
	}
	hasChild, err := rd.Bool()
	if err != nil {
		return err
	}
	if !hasChild {
		r.Child = nil
		return nil
	}
	if r.Child == nil {
		r.Child = &benchRecord{}
	}
	return r.Child.DecodeOrc(rd)
}

func benchInput() *benchRecord {
	return &benchRecord{
		U8:    8,
		U16:   16,
		U32:   32,
		U64:   64,
		I8:    -8,
		I16:   -16,
		I32:   -32,
		I64:   -64,
		F32:   2.5,
		F64:   3.14159,
		Flag:  true,
		Name:  "bench-record",
		Blob:  []byte{0xDE, 0xAD, 0xBE, 0xEF},
		Tags:  []string{"alpha", "beta"},
		Raw:   [4]byte{1, 2, 3, 4},
		Child: &benchRecord{Name: "child"},
	}
}

func benchPayload(b *testing.B) []byte {
	b.Helper()
	data, err := orc.Codec.Encode(context.Background(), benchInput())
	if err != nil {
		b.Fatal(err)
	}
	return data
}

func BenchmarkCodecEncode(b *testing.B) {
	ctx := context.Background()
	in := benchInput()
	if _, err := orc.Codec.Encode(ctx, in); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := orc.Codec.Encode(ctx, in); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodecDecode(b *testing.B) {
	ctx := context.Background()
	data := benchPayload(b)
	out := &benchRecord{}
	if err := orc.Codec.Decode(ctx, data, out); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := orc.Codec.Decode(ctx, data, out); err != nil {
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
	for b.Loop() {
		if err := orc.Codec.Decode(ctx, data, out); err == nil {
			b.Fatal("expected magic mismatch error")
		}
	}
}

// BenchmarkWriter measures raw Writer throughput with a warm reused buffer, outside
// the codec's pool and output-copy overhead.
func BenchmarkWriter(b *testing.B) {
	in := benchInput()
	w := orc.NewWriter(256)
	b.ReportAllocs()
	for b.Loop() {
		w.Reset()
		if err := in.EncodeOrc(w); err != nil {
			b.Fatal(err)
		}
		if w.Len() != len(w.Bytes()) {
			b.Fatal("length mismatch")
		}
	}
}

// BenchmarkReaderBytes measures raw Reader throughput in direct byte-slice mode,
// outside the codec's pool and magic-check overhead.
func BenchmarkReaderBytes(b *testing.B) {
	payload := benchPayload(b)[len(magic):]
	rd := orc.NewReader(nil)
	out := &benchRecord{}
	b.ReportAllocs()
	for b.Loop() {
		rd.ResetBytes(payload)
		if err := out.DecodeOrc(rd); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReaderStream measures Reader throughput in io.Reader mode, which pays
// io.ReadFull per fixed-size value and an intermediate buffer per string.
func BenchmarkReaderStream(b *testing.B) {
	payload := benchPayload(b)[len(magic):]
	br := bytes.NewReader(payload)
	rd := orc.NewReader(br)
	out := &benchRecord{}
	b.ReportAllocs()
	for b.Loop() {
		br.Reset(payload)
		rd.Reset(br)
		if err := out.DecodeOrc(rd); err != nil {
			b.Fatal(err)
		}
	}
}
