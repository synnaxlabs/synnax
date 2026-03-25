// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"context"
	"encoding/binary"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

// These tests encode data with one codec, run the schema resolver, and decode
// with the updated codec. They prove the resolver's output matches what the
// codec expects.

// simpleEntry is a type with a few fields for testing codec round-trips.
type simpleEntry struct {
	ID   int32  `msgpack:"id"`
	Name string `msgpack:"name"`
}

func (e simpleEntry) GorpKey() int32    { return e.ID }
func (e simpleEntry) SetOptions() []any { return nil }

// simpleEntryV2 adds a Description field.
type simpleEntryV2 struct {
	ID          int32  `msgpack:"id"`
	Name        string `msgpack:"name"`
	Description string `msgpack:"description"`
}

func (e simpleEntryV2) GorpKey() int32    { return e.ID }
func (e simpleEntryV2) SetOptions() []any { return nil }

// simpleCodecV1 is a binary codec for simpleEntry (v1: id + name).
type simpleCodecV1 struct{}

func (simpleCodecV1) Encode(_ context.Context, value any) ([]byte, error) {
	e := value.(simpleEntry)
	var buf []byte
	buf = binary.BigEndian.AppendUint32(buf, uint32(e.ID))
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(e.Name)))
	buf = append(buf, e.Name...)
	return buf, nil
}

func (simpleCodecV1) Decode(_ context.Context, data []byte, value any) error {
	e := value.(*simpleEntry)
	if len(data) < 4 {
		return nil
	}
	e.ID = int32(binary.BigEndian.Uint32(data[:4]))
	data = data[4:]
	if len(data) < 4 {
		return nil
	}
	n := binary.BigEndian.Uint32(data[:4])
	data = data[4:]
	if len(data) < int(n) {
		return nil
	}
	e.Name = string(data[:n])
	return nil
}

func (c simpleCodecV1) EncodeStream(_ context.Context, w io.Writer, v any) error {
	b, err := c.Encode(context.Background(), v)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (c simpleCodecV1) DecodeStream(_ context.Context, r io.Reader, v any) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return c.Decode(context.Background(), data, v)
}

// simpleCodecV2 is a binary codec for simpleEntryV2 (v2: id + name + description).
type simpleCodecV2 struct{}

func (simpleCodecV2) Encode(_ context.Context, value any) ([]byte, error) {
	e := value.(simpleEntryV2)
	var buf []byte
	buf = binary.BigEndian.AppendUint32(buf, uint32(e.ID))
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(e.Name)))
	buf = append(buf, e.Name...)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(e.Description)))
	buf = append(buf, e.Description...)
	return buf, nil
}

func (simpleCodecV2) Decode(_ context.Context, data []byte, value any) error {
	e := value.(*simpleEntryV2)
	if len(data) < 4 {
		return nil
	}
	e.ID = int32(binary.BigEndian.Uint32(data[:4]))
	data = data[4:]
	if len(data) < 4 {
		return nil
	}
	n := binary.BigEndian.Uint32(data[:4])
	data = data[4:]
	if len(data) < int(n) {
		return nil
	}
	e.Name = string(data[:n])
	data = data[n:]
	if len(data) < 4 {
		return nil
	}
	n2 := binary.BigEndian.Uint32(data[:4])
	data = data[4:]
	if len(data) < int(n2) {
		return nil
	}
	e.Description = string(data[:n2])
	return nil
}

func (c simpleCodecV2) EncodeStream(_ context.Context, w io.Writer, v any) error {
	b, err := c.Encode(context.Background(), v)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (c simpleCodecV2) DecodeStream(_ context.Context, r io.Reader, v any) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return c.Decode(context.Background(), data, v)
}

var v1Layout = []gorp.FieldLayout{
	{Name: "id", Encoding: gorp.EncodingInt32},
	{Name: "name", Encoding: gorp.EncodingString},
}

var v2Layout = []gorp.FieldLayout{
	{Name: "id", Encoding: gorp.EncodingInt32},
	{Name: "name", Encoding: gorp.EncodingString},
	{Name: "description", Encoding: gorp.EncodingString},
}

var _ = Describe("Schema Resolution Integration", func() {
	ctx := context.Background()

	Describe("Full pipeline: encode v1 -> resolve -> decode v2", func() {
		It("Should add a field and decode correctly", func() {
			// Encode with v1 codec
			v1Data := MustSucceed(simpleCodecV1{}.Encode(ctx, simpleEntry{ID: 42, Name: "test"}))

			// Resolve from v1 layout to v2 layout
			resolved := MustSucceed(gorp.Resolve(v1Data, v1Layout, v2Layout))

			// Decode with v2 codec
			var result simpleEntryV2
			Expect(simpleCodecV2{}.Decode(ctx, resolved, &result)).To(Succeed())
			Expect(result.ID).To(Equal(int32(42)))
			Expect(result.Name).To(Equal("test"))
			Expect(result.Description).To(Equal(""))
		})
	})

	Describe("SchemaEvolution migration end-to-end", func() {
		It("Should migrate entries through OpenTable", func() {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()

			// Write v1 entries using v1 codec under the simpleEntryV2 prefix
			// (simulating what would happen in production where the type name
			// stays the same across versions).
			v1Codec := simpleCodecV1{}
			// v1Codec.Encode expects simpleEntry, but WrapWriter passes simpleEntryV2.
			// Since the fields overlap (ID, Name), we write raw bytes instead.
			prefix := "__gorp__//simpleEntryV2"
			for _, e := range []simpleEntry{{ID: 1, Name: "alice"}, {ID: 2, Name: "bob"}} {
				data := MustSucceed(v1Codec.Encode(ctx, e))
				key := make([]byte, len(prefix)+4)
				copy(key, prefix)
				binary.BigEndian.PutUint32(key[len(prefix):], uint32(e.ID))
				Expect(testDB.Set(ctx, key, data)).To(Succeed())
			}

			// Open table with v2 codec and schema evolution migration.
			v2Codec := simpleCodecV2{}
			MustSucceed(gorp.OpenTable[int32, simpleEntryV2](ctx, gorp.TableConfig[simpleEntryV2]{
				DB:    testDB,
				Codec: v2Codec,
				Migrations: []gorp.Migration{
					gorp.NewSchemaEvolution[simpleEntryV2](
						"v2_add_description",
						v1Layout, v2Layout,
						v2Codec,
						func(_ context.Context, old simpleEntryV2) (simpleEntryV2, error) {
							old.Description = "migrated"
							return old, nil
						},
					),
				},
			}))

			// Read with v2 codec and verify
			r := gorp.WrapReader[int32, simpleEntryV2](testDB, v2Codec)
			alice := MustSucceed(r.Get(ctx, 1))
			Expect(alice.Name).To(Equal("alice"))
			Expect(alice.Description).To(Equal("migrated"))

			bob := MustSucceed(r.Get(ctx, 2))
			Expect(bob.Name).To(Equal("bob"))
			Expect(bob.Description).To(Equal("migrated"))
		})
	})
})
