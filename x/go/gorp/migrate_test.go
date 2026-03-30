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
	stdbinary "encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/graph"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type entryV1 struct {
	ID   int32  `msgpack:"id"`
	Data string `msgpack:"data"`
}

func (e entryV1) GorpKey() int32    { return e.ID }
func (e entryV1) SetOptions() []any { return nil }

type entryV2 struct {
	ID          int32  `msgpack:"id"`
	Data        string `msgpack:"data"`
	Description string `msgpack:"description"`
}

func (e entryV2) GorpKey() int32    { return e.ID }
func (e entryV2) SetOptions() []any { return nil }

type jsonEntry struct {
	ID   int32  `json:"id"`
	Data string `json:"data"`
}

func (e jsonEntry) GorpKey() int32    { return e.ID }
func (e jsonEntry) SetOptions() []any { return nil }

type jsonEntryCodec struct{}

func (jsonEntryCodec) Encode(_ context.Context, value any) ([]byte, error) {
	return json.Marshal(value)
}

func (jsonEntryCodec) Decode(_ context.Context, data []byte, value any) error {
	return json.Unmarshal(data, value)
}

func (c jsonEntryCodec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	b, err := c.Encode(context.Background(), value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (c jsonEntryCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return c.Decode(context.Background(), data, value)
}

var _ binary.Codec = jsonEntryCodec{}

type failMarshalCodec struct{}

func (failMarshalCodec) Encode(_ context.Context, _ any) ([]byte, error) {
	return nil, errors.New("marshal failed")
}

func (failMarshalCodec) Decode(_ context.Context, data []byte, value any) error {
	return json.Unmarshal(data, value)
}

func (c failMarshalCodec) EncodeStream(_ context.Context, _ io.Writer, _ any) error {
	return errors.New("marshal failed")
}

func (c failMarshalCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return c.Decode(context.Background(), data, value)
}

var _ binary.Codec = failMarshalCodec{}

type migrationDepProvider interface {
	GetSuffix() string
}

type migrationDepProviderImpl string

func (p migrationDepProviderImpl) GetSuffix() string { return string(p) }

// mapEntry mimics production types like Schematic that have a MsgpackEncodedJSON field.
type mapEntry struct {
	ID   int32                     `msgpack:"id"`
	Name string                    `msgpack:"name"`
	Data binary.MsgpackEncodedJSON `msgpack:"data"`
}

func (e mapEntry) GorpKey() int32    { return e.ID }
func (e mapEntry) SetOptions() []any { return nil }

// structpbCodec mimics the production protobuf codecs (e.g. schematicpb.SchematicCodec)
// by marshaling the Data field through structpb.NewStruct, which is the exact path
// that production code takes.
type structpbCodec struct{}

func (structpbCodec) Encode(_ context.Context, value any) ([]byte, error) {
	e := value.(mapEntry)
	dataVal, err := structpb.NewStruct(e.Data)
	if err != nil {
		return nil, err
	}
	pb := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"id":   structpb.NewNumberValue(float64(e.ID)),
			"name": structpb.NewStringValue(e.Name),
			"data": structpb.NewStructValue(dataVal),
		},
	}
	return proto.Marshal(pb)
}

func (structpbCodec) Decode(_ context.Context, data []byte, value any) error {
	pb := &structpb.Struct{}
	if err := proto.Unmarshal(data, pb); err != nil {
		return err
	}
	m := pb.AsMap()
	e := value.(*mapEntry)
	e.ID = int32(m["id"].(float64))
	e.Name = m["name"].(string)
	if d, ok := m["data"]; ok && d != nil {
		e.Data = d.(map[string]any)
	}
	return nil
}

func (c structpbCodec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	b, err := c.Encode(context.Background(), value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (c structpbCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return c.Decode(context.Background(), data, value)
}

var _ binary.Codec = structpbCodec{}

var _ = Describe("Gorp", func() {
	Describe("Codec", func() {
		It("Should use Codec.Marshal when a codec is provided to the table", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			tbl := MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
				DB:    testDB,
				Codec: jsonEntryCodec{},
			}))
			e := jsonEntry{ID: 1, Data: "hello"}
			Expect(tbl.NewCreate().Entry(&e).Exec(ctx, testDB)).To(Succeed())
			var result jsonEntry
			Expect(tbl.NewRetrieve().WhereKeys(1).Entry(&result).Exec(ctx, testDB)).To(Succeed())
			Expect(result.ID).To(Equal(int32(1)))
			Expect(result.Data).To(Equal("hello"))
		})

		It("Should fall back to DB codec when no table codec is provided", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entry](testDB, testDB)
			Expect(w.Set(ctx, entry{ID: 1, Data: "hello"})).To(Succeed())
			r := gorp.WrapReader[int32, entry](testDB, testDB)
			result := MustSucceed(r.Get(ctx, 1))
			Expect(result.ID).To(Equal(int32(1)))
			Expect(result.Data).To(Equal("hello"))
		})

		It("Should use Codec.Unmarshal in Iterator.Value via table reader", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			tbl := MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
				DB:    testDB,
				Codec: jsonEntryCodec{},
			}))
			e := jsonEntry{ID: 1, Data: "iter"}
			Expect(tbl.NewCreate().Entry(&e).Exec(ctx, testDB)).To(Succeed())
			var result jsonEntry
			Expect(tbl.NewRetrieve().WhereKeys(1).Entry(&result).Exec(ctx, testDB)).To(Succeed())
			Expect(result.ID).To(Equal(int32(1)))
			Expect(result.Data).To(Equal("iter"))
		})
	})

	Describe("Migration Interface", func() {
		Describe("TypedMigration", func() {
			It("Should transform entries from one schema to another", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
				Expect(w.Set(ctx, entryV1{ID: 2, Data: "two"})).To(Succeed())
				migration := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"add_suffix",
					nil, nil,
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_migrated"}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("one_migrated"))
				Expect(MustSucceed(r.Get(ctx, 2)).Data).To(Equal("two_migrated"))
			})

			It("Should call PostMigrateFunc after auto migration", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
				migration := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"post_transform",
					nil, nil,
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: "post:" + old.Data}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("post:one"))
			})

			It("Should handle cross-type migration with manually seeded data", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				codec := &binary.MsgPackCodec{}
				prefix := "__gorp__//entryV2"
				for _, e := range []entryV1{{ID: 1, Data: "one"}, {ID: 2, Data: "two"}} {
					data := MustSucceed(codec.Encode(ctx, e))
					key := make([]byte, len(prefix)+4)
					copy(key, prefix)
					stdbinary.BigEndian.PutUint32(key[len(prefix):], uint32(e.ID))
					Expect(testDB.Set(ctx, key, data)).To(Succeed())
				}
				migration := gorp.NewTypedMigration[int32, int32, entryV1, entryV2](
					"v1_to_v2",
					nil, nil,
					func(_ context.Context, old entryV1) (entryV2, error) {
						return entryV2{
							ID:          old.ID,
							Data:        old.Data,
							Description: "migrated:" + old.Data,
						}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV2](ctx, gorp.TableConfig[entryV2]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV2](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Description).To(Equal("migrated:one"))
				Expect(MustSucceed(r.Get(ctx, 2)).Description).To(Equal("migrated:two"))
			})
		})

		Describe("RawMigration", func() {
			It("Should provide a working gorp.Tx for read/write", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "raw"})).To(Succeed())
				migration := gorp.NewRawMigration(
					"raw_transform",
					func(ctx context.Context, tx gorp.Tx) error {
						r := gorp.WrapReader[int32, entryV1](tx, tx)
						e := MustSucceed(r.Get(ctx, 1))
						e.Data = "raw_migrated"
						w := gorp.WrapWriter[int32, entryV1](tx, tx)
						return w.Set(ctx, e)
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("raw_migrated"))
			})
		})

		Describe("Version tracking", func() {
			It("Should store applied migration names", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				migration := gorp.NewRawMigration(
					"noop",
					func(context.Context, gorp.Tx) error { return nil },
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				versionKey := []byte("__gorp_migration__//entryV1")
				b, closer := MustSucceed2(testDB.Get(ctx, versionKey))
				Expect(closer.Close()).To(Succeed())
				Expect(string(b)).To(Equal("noop"))
			})

			It("Should skip already-completed migrations on re-run", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				executionCount := 0
				migration := gorp.NewRawMigration(
					"counted",
					func(context.Context, gorp.Tx) error {
						executionCount++
						return nil
					},
				)
				cfg := gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				Expect(executionCount).To(Equal(1))
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				Expect(executionCount).To(Equal(1))
			})

			It("Should only run new migrations after partial completion", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var executed []string
				m1 := gorp.NewRawMigration("first", func(context.Context, gorp.Tx) error {
					executed = append(executed, "first")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1},
				}))
				Expect(executed).To(Equal([]string{"first"}))
				m2 := gorp.NewRawMigration("second", func(context.Context, gorp.Tx) error {
					executed = append(executed, "second")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				}))
				Expect(executed).To(Equal([]string{"first", "second"}))
			})
		})

		Describe("Sequential execution / chaining", func() {
			It("Should chain two TypedMigrations sequentially", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "chain"})).To(Succeed())
				m1 := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"add_suffix",
					nil, nil,
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_v2"}, nil
					},
				)
				m2 := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"add_suffix_2",
					nil, nil,
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_v3"}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("chain_v2_v3"))
			})

			It("Should chain a TypedMigration with a RawMigration", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "mixed"})).To(Succeed())
				m1 := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"typed_transform",
					nil, nil,
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_typed"}, nil
					},
				)
				m2 := gorp.NewRawMigration("raw_update", func(ctx context.Context, tx gorp.Tx) error {
					r := gorp.WrapReader[int32, entryV1](tx, tx)
					e := MustSucceed(r.Get(ctx, 1))
					e.Data = e.Data + "_raw"
					w := gorp.WrapWriter[int32, entryV1](tx, tx)
					return w.Set(ctx, e)
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("mixed_typed_raw"))
			})
		})

		Describe("Error handling", func() {
			It("Should not commit when a migration fails", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "original"})).To(Succeed())
				migration := gorp.NewRawMigration(
					"failing",
					func(context.Context, gorp.Tx) error {
						return errors.New("migration error")
					},
				)
				_, err := gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failing"))
				versionKey := []byte("__gorp_migration__//entryV1")
				Expect(testDB.Get(ctx, versionKey)).Error().To(MatchError(query.ErrNotFound))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("original"))
			})
		})

		Describe("CodecTransitionMigration", func() {
			It("Should re-encode entries from default codec to custom codec", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB, testDB)
				Expect(w.Set(ctx, jsonEntry{ID: 1, Data: "one"})).To(Succeed())
				Expect(w.Set(ctx, jsonEntry{ID: 2, Data: "two"})).To(Succeed())
				tbl := MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
					DB:    testDB,
					Codec: jsonEntryCodec{},
					Migrations: []gorp.Migration{
						gorp.NewCodecTransition[int32, jsonEntry]("msgpack_to_json", jsonEntryCodec{}),
					},
				}))
				var r1, r2 jsonEntry
				Expect(tbl.NewRetrieve().WhereKeys(1).Entry(&r1).Exec(ctx, testDB)).To(Succeed())
				Expect(r1.ID).To(Equal(int32(1)))
				Expect(r1.Data).To(Equal("one"))
				Expect(tbl.NewRetrieve().WhereKeys(2).Entry(&r2).Exec(ctx, testDB)).To(Succeed())
				Expect(r2.Data).To(Equal("two"))
			})

			It("Should produce bytes in the target codec format", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB, testDB)
				Expect(w.Set(ctx, jsonEntry{ID: 1, Data: "verify"})).To(Succeed())
				MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
					DB:    testDB,
					Codec: jsonEntryCodec{},
					Migrations: []gorp.Migration{
						gorp.NewCodecTransition[int32, jsonEntry]("msgpack_to_json", jsonEntryCodec{}),
					},
				}))
				prefix := "__gorp__//jsonEntry"
				key := make([]byte, len(prefix)+4)
				copy(key, prefix)
				stdbinary.BigEndian.PutUint32(key[len(prefix):], 1)
				raw, closer := MustSucceed2(testDB.Get(ctx, key))
				Expect(closer.Close()).To(Succeed())
				var parsed map[string]interface{}
				Expect(json.Unmarshal(raw, &parsed)).To(Succeed())
				Expect(parsed["data"]).To(Equal("verify"))
			})

			It("Should be a no-op on second run", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB, testDB)
				Expect(w.Set(ctx, jsonEntry{ID: 1, Data: "hello"})).To(Succeed())
				cfg := gorp.TableConfig[jsonEntry]{
					DB:    testDB,
					Codec: jsonEntryCodec{},
					Migrations: []gorp.Migration{
						gorp.NewCodecTransition[int32, jsonEntry]("msgpack_to_json", jsonEntryCodec{}),
					},
				}
				MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, cfg))
				tbl := MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, cfg))
				var result jsonEntry
				Expect(tbl.NewRetrieve().WhereKeys(1).Entry(&result).Exec(ctx, testDB)).To(Succeed())
				Expect(result.Data).To(Equal("hello"))
			})

			It("Should handle empty DB", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
					DB:    testDB,
					Codec: jsonEntryCodec{},
					Migrations: []gorp.Migration{
						gorp.NewCodecTransition[int32, jsonEntry]("msgpack_to_json", jsonEntryCodec{}),
					},
				}))
			})

			It("Should migrate many entries correctly", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB, testDB)
				for i := int32(0); i < 100; i++ {
					Expect(w.Set(ctx, jsonEntry{ID: i, Data: fmt.Sprintf("entry_%d", i)})).To(Succeed())
				}
				tbl := MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
					DB:    testDB,
					Codec: jsonEntryCodec{},
					Migrations: []gorp.Migration{
						gorp.NewCodecTransition[int32, jsonEntry]("msgpack_to_json", jsonEntryCodec{}),
					},
				}))
				for i := int32(0); i < 100; i++ {
					var result jsonEntry
					Expect(tbl.NewRetrieve().WhereKeys(i).Entry(&result).Exec(ctx, testDB)).To(Succeed())
					Expect(result.Data).To(Equal(fmt.Sprintf("entry_%d", i)))
				}
			})

			It("Should return an error when source data cannot be decoded", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				prefix := "__gorp__//jsonEntry"
				key := make([]byte, len(prefix)+4)
				copy(key, prefix)
				stdbinary.BigEndian.PutUint32(key[len(prefix):], 1)
				Expect(testDB.Set(ctx, key, []byte("not valid msgpack \xff\xfe"))).To(Succeed())
				Expect(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
					DB:    testDB,
					Codec: jsonEntryCodec{},
					Migrations: []gorp.Migration{
						gorp.NewCodecTransition[int32, jsonEntry]("msgpack_to_json", jsonEntryCodec{}),
					},
				})).Error().To(MatchError(ContainSubstring("failed to decode")))
			})

			It("Should return an error when target codec fails to marshal", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB, testDB)
				Expect(w.Set(ctx, jsonEntry{ID: 1, Data: "fail"})).To(Succeed())
				Expect(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
					DB:    testDB,
					Codec: failMarshalCodec{},
					Migrations: []gorp.Migration{
						gorp.NewCodecTransition[int32, jsonEntry]("to_failing", failMarshalCodec{}),
					},
				})).Error().To(MatchError(ContainSubstring("marshal failed")))
			})

			It("Should work in a migration chain with a preceding RawMigration", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB, testDB)
				Expect(w.Set(ctx, jsonEntry{ID: 1, Data: "chain"})).To(Succeed())
				m1 := gorp.NewRawMigration("raw_update", func(ctx context.Context, tx gorp.Tx) error {
					r := gorp.WrapReader[int32, jsonEntry](tx, tx)
					e := MustSucceed(r.Get(ctx, 1))
					e.Data = e.Data + "_raw"
					w := gorp.WrapWriter[int32, jsonEntry](tx, tx)
					return w.Set(ctx, e)
				})
				m2 := gorp.NewCodecTransition[int32, jsonEntry]("msgpack_to_json", jsonEntryCodec{})
				tbl := MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
					DB:         testDB,
					Codec:      jsonEntryCodec{},
					Migrations: []gorp.Migration{m1, m2},
				}))
				var result jsonEntry
				Expect(tbl.NewRetrieve().WhereKeys(1).Entry(&result).Exec(ctx, testDB)).To(Succeed())
				Expect(result.Data).To(Equal("chain_raw"))
			})

			It("Should not commit when migration fails", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB, testDB)
				Expect(w.Set(ctx, jsonEntry{ID: 1, Data: "original"})).To(Succeed())
				Expect(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
					DB:    testDB,
					Codec: failMarshalCodec{},
					Migrations: []gorp.Migration{
						gorp.NewCodecTransition[int32, jsonEntry]("to_failing", failMarshalCodec{}),
					},
				})).Error().To(HaveOccurred())
				Expect(testDB.Get(ctx, []byte("__gorp_migration__//jsonEntry"))).
					Error().To(MatchError(query.ErrNotFound))
				r := gorp.WrapReader[int32, jsonEntry](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("original"))
			})

			It("Should re-encode MsgpackEncodedJSON fields through structpb", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, mapEntry](testDB, testDB)
				Expect(w.Set(ctx, mapEntry{
					ID:   1,
					Name: "nested_test",
					Data: binary.MsgpackEncodedJSON{
						"string_val": "hello",
						"int_val":    42,
						"float_val":  3.14,
						"bool_val":   true,
						"null_val":   nil,
						"nested": map[string]any{
							"level2_string": "deep",
							"level2_int":    99,
							"level3": map[string]any{
								"key": "deeply_nested",
							},
						},
						"array": []any{"a", "b", "c"},
						"mixed_array": []any{
							1, "two", true, nil,
							map[string]any{"inner": "map"},
						},
					},
				})).To(Succeed())
				codec := structpbCodec{}
				tbl := MustSucceed(gorp.OpenTable[int32, mapEntry](ctx, gorp.TableConfig[mapEntry]{
					DB:    testDB,
					Codec: codec,
					Migrations: []gorp.Migration{
						gorp.NewCodecTransition[int32, mapEntry]("msgpack_to_structpb", codec),
					},
				}))
				var result mapEntry
				Expect(tbl.NewRetrieve().WhereKeys(1).Entry(&result).Exec(ctx, testDB)).To(Succeed())
				Expect(result.Name).To(Equal("nested_test"))
				Expect(result.Data["string_val"]).To(Equal("hello"))
				Expect(result.Data["bool_val"]).To(Equal(true))
				Expect(result.Data["null_val"]).To(BeNil())
				nested := result.Data["nested"].(map[string]any)
				Expect(nested["level2_string"]).To(Equal("deep"))
				level3 := nested["level3"].(map[string]any)
				Expect(level3["key"]).To(Equal("deeply_nested"))
				arr := result.Data["array"].([]any)
				Expect(arr).To(HaveLen(3))
				Expect(arr[0]).To(Equal("a"))
			})
		})

		Describe("Zero-migration case", func() {
			It("Should run key re-encoding only when no migrations are provided", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "no_migration"})).To(Succeed())
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB: testDB,
				}))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("no_migration"))
			})
		})

		Describe("Idempotent key migration", func() {
			It("Should run key re-encoding even when versioned migrations are at latest", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				migration := gorp.NewRawMigration(
					"noop",
					func(context.Context, gorp.Tx) error { return nil },
				)
				cfg := gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 5, Data: "post_migration"})).To(Succeed())
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 5)).Data).To(Equal("post_migration"))
			})
		})

		Describe("WithDependencies", func() {
			It("Should order migrations respecting declared dependencies", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var order []string
				m1 := gorp.NewRawMigration("alpha", func(context.Context, gorp.Tx) error {
					order = append(order, "alpha")
					return nil
				})
				m2 := gorp.WithDependencies(
					gorp.NewRawMigration("beta", func(context.Context, gorp.Tx) error {
						order = append(order, "beta")
						return nil
					}),
					"alpha",
				)
				m3 := gorp.WithDependencies(
					gorp.NewRawMigration("gamma", func(context.Context, gorp.Tx) error {
						order = append(order, "gamma")
						return nil
					}),
					"beta",
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m3, m2, m1},
				}))
				Expect(order).To(Equal([]string{"alpha", "beta", "gamma"}))
			})

			It("Should treat already-applied dependencies as satisfied", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var order []string
				m1 := gorp.NewRawMigration("first", func(context.Context, gorp.Tx) error {
					order = append(order, "first")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1},
				}))
				Expect(order).To(Equal([]string{"first"}))
				m2 := gorp.WithDependencies(
					gorp.NewRawMigration("second", func(context.Context, gorp.Tx) error {
						order = append(order, "second")
						return nil
					}),
					"first",
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				}))
				Expect(order).To(Equal([]string{"first", "second"}))
			})

			It("Should detect cyclic dependencies", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				m1 := gorp.WithDependencies(
					gorp.NewRawMigration("a", func(context.Context, gorp.Tx) error { return nil }),
					"b",
				)
				m2 := gorp.WithDependencies(
					gorp.NewRawMigration("b", func(context.Context, gorp.Tx) error { return nil }),
					"a",
				)
				_, err := gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("cyclic dependency")))
			})

			It("Should return error for missing dependency", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				m1 := gorp.WithDependencies(
					gorp.NewRawMigration("orphan", func(context.Context, gorp.Tx) error { return nil }),
					"nonexistent",
				)
				_, err := gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1},
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(graph.ErrMissingDependency))
				Expect(err).To(MatchError(ContainSubstring("nonexistent")))
			})

			It("Should work with migrations that do not declare dependencies", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var order []string
				m1 := gorp.NewRawMigration("plain_a", func(context.Context, gorp.Tx) error {
					order = append(order, "plain_a")
					return nil
				})
				m2 := gorp.NewRawMigration("plain_b", func(context.Context, gorp.Tx) error {
					order = append(order, "plain_b")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				}))
				Expect(order).To(Equal([]string{"plain_a", "plain_b"}))
			})

			It("Should preserve insertion order for migrations without dependencies "+
				"when mixed with dependency-declaring migrations", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var order []string
				m1 := gorp.NewRawMigration("no_dep", func(context.Context, gorp.Tx) error {
					order = append(order, "no_dep")
					return nil
				})
				m2 := gorp.WithDependencies(
					gorp.NewRawMigration("has_dep", func(context.Context, gorp.Tx) error {
						order = append(order, "has_dep")
						return nil
					}),
					"no_dep",
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m2, m1},
				}))
				Expect(order).To(Equal([]string{"no_dep", "has_dep"}))
			})
		})

		Describe("MigrationDep", func() {
			It("Should inject and retrieve a dependency via context", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "original"})).To(Succeed())
				type LookupService struct {
					Suffix string
				}
				depCtx := gorp.WithMigrationDep[*LookupService](ctx, &LookupService{Suffix: "_enriched"})
				migration := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"enrich",
					nil, nil,
					func(ctx context.Context, old entryV1) (entryV1, error) {
						svc := gorp.MigrationDep[*LookupService](ctx)
						return entryV1{ID: old.ID, Data: old.Data + svc.Suffix}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](depCtx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("original_enriched"))
			})

			It("Should support multiple dependencies of different types", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "base"})).To(Succeed())
				type PrefixService struct {
					Prefix string
				}
				type SuffixService struct {
					Suffix string
				}
				depCtx := gorp.WithMigrationDep[*PrefixService](ctx, &PrefixService{Prefix: "pre_"})
				depCtx = gorp.WithMigrationDep[*SuffixService](depCtx, &SuffixService{Suffix: "_post"})
				migration := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"wrap",
					nil, nil,
					func(ctx context.Context, old entryV1) (entryV1, error) {
						pre := gorp.MigrationDep[*PrefixService](ctx)
						suf := gorp.MigrationDep[*SuffixService](ctx)
						return entryV1{
							ID:   old.ID,
							Data: pre.Prefix + old.Data + suf.Suffix,
						}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](depCtx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("pre_base_post"))
			})

			It("Should panic when a required dependency is missing", func(ctx SpecContext) {
				type MissingService struct{}
				Expect(func() {
					gorp.MigrationDep[*MissingService](ctx)
				}).To(PanicWith(ContainSubstring("missing migration dependency")))
			})

			It("Should return false from MigrationDepOpt when dependency is missing", func(ctx SpecContext) {
				type OptionalService struct{}
				_, ok := gorp.MigrationDepOpt[*OptionalService](ctx)
				Expect(ok).To(BeFalse())
			})

			It("Should return the dependency from MigrationDepOpt when present", func(ctx SpecContext) {
				type OptionalService struct {
					Value string
				}
				depCtx := gorp.WithMigrationDep[*OptionalService](ctx, &OptionalService{Value: "here"})
				svc, ok := gorp.MigrationDepOpt[*OptionalService](depCtx)
				Expect(ok).To(BeTrue())
				Expect(svc.Value).To(Equal("here"))
			})

			It("Should work with RawMigration", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "raw"})).To(Succeed())
				type Renamer struct {
					NewData string
				}
				depCtx := gorp.WithMigrationDep[*Renamer](ctx, &Renamer{NewData: "renamed"})
				migration := gorp.NewRawMigration(
					"raw_with_dep",
					func(ctx context.Context, tx gorp.Tx) error {
						renamer := gorp.MigrationDep[*Renamer](ctx)
						r := gorp.WrapReader[int32, entryV1](tx, tx)
						e := MustSucceed(r.Get(ctx, 1))
						e.Data = renamer.NewData
						w := gorp.WrapWriter[int32, entryV1](tx, tx)
						return w.Set(ctx, e)
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](depCtx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("renamed"))
			})

			It("Should work with interface types", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "iface"})).To(Succeed())
				depCtx := gorp.WithMigrationDep[migrationDepProvider](
					ctx, migrationDepProviderImpl("_from_iface"),
				)
				migration := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"iface_dep",
					nil, nil,
					func(ctx context.Context, old entryV1) (entryV1, error) {
						dp := gorp.MigrationDep[migrationDepProvider](ctx)
						return entryV1{ID: old.ID, Data: old.Data + dp.GetSuffix()}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](depCtx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB, testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("iface_from_iface"))
			})
		})

		Describe("EntryCounter", func() {
			It("Should track entries processed by TypedMigration", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				for i := int32(0); i < 5; i++ {
					Expect(w.Set(ctx, entryV1{ID: i, Data: "x"})).To(Succeed())
				}
				migration := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"count_test",
					nil, nil,
					func(_ context.Context, old entryV1) (entryV1, error) {
						return old, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				ec, ok := migration.(gorp.EntryCounter)
				Expect(ok).To(BeTrue())
				Expect(ec.EntriesProcessed()).To(Equal(5))
			})

			It("Should track entries processed by CodecTransition", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB, testDB)
				for i := int32(0); i < 3; i++ {
					Expect(w.Set(ctx, jsonEntry{ID: i, Data: "x"})).To(Succeed())
				}
				migration := gorp.NewCodecTransition[int32, jsonEntry]("codec_count", jsonEntryCodec{})
				MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
					DB:         testDB,
					Codec:      jsonEntryCodec{},
					Migrations: []gorp.Migration{migration},
				}))
				ec, ok := migration.(gorp.EntryCounter)
				Expect(ok).To(BeTrue())
				Expect(ec.EntriesProcessed()).To(Equal(3))
			})
		})

		Describe("Error context", func() {
			It("Should include entry key in transform error", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 42, Data: "bad"})).To(Succeed())
				migration := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"fail_transform",
					nil, nil,
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{}, errors.New("transform broke")
					},
				)
				_, err := gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("42"))
				Expect(err.Error()).To(ContainSubstring("transform"))
			})

			It("Should include raw key in decode error", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				prefix := "__gorp__//entryV1"
				key := make([]byte, len(prefix)+4)
				copy(key, prefix)
				stdbinary.BigEndian.PutUint32(key[len(prefix):], 1)
				Expect(testDB.Set(ctx, key, []byte("not valid msgpack"))).To(Succeed())
				migration := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"fail_decode",
					nil, nil,
					func(_ context.Context, old entryV1) (entryV1, error) {
						return old, nil
					},
				)
				_, err := gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("decode"))
			})
		})

		Describe("Log output", func() {
			newObservedIns := func() (alamos.Instrumentation, *observer.ObservedLogs) {
				core, logs := observer.New(zapcore.DebugLevel)
				logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
					ZapLogger: zap.New(core),
				}))
				return alamos.New("test", alamos.WithLogger(logger)), logs
			}

			It("Should log starting and completion messages for pending migrations", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB, testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "x"})).To(Succeed())
				ins, logs := newObservedIns()
				migration := gorp.NewTypedMigration[int32, int32, entryV1, entryV1](
					"test_migration",
					nil, nil,
					func(_ context.Context, old entryV1) (entryV1, error) {
						return old, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:              testDB,
					Migrations:      []gorp.Migration{migration},
					Instrumentation: ins,
				}))
				starting := logs.FilterMessage("starting migrations")
				Expect(starting.Len()).To(Equal(1))
				Expect(starting.All()[0].ContextMap()["pending"]).To(BeNumerically("==", 1))

				complete := logs.FilterMessage("migration complete")
				Expect(complete.Len()).To(Equal(1))
				Expect(complete.All()[0].ContextMap()["migration"]).To(Equal("test_migration"))
				Expect(complete.All()[0].ContextMap()["entries"]).To(BeNumerically("==", 1))

				done := logs.FilterMessage("migrations complete")
				Expect(done.Len()).To(Equal(1))
				Expect(done.All()[0].ContextMap()["migrations"]).To(BeNumerically("==", 1))
			})

			It("Should not log when all migrations are already applied", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				migration := gorp.NewRawMigration(
					"noop",
					func(context.Context, gorp.Tx) error { return nil },
				)
				cfg := gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				ins, logs := newObservedIns()
				cfg.Instrumentation = ins
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				Expect(logs.Len()).To(Equal(0))
			})

			It("Should log already applied migrations at debug level", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				m1 := gorp.NewRawMigration("first", func(context.Context, gorp.Tx) error { return nil })
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1},
				}))
				ins, logs := newObservedIns()
				m2 := gorp.NewRawMigration("second", func(context.Context, gorp.Tx) error { return nil })
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:              testDB,
					Migrations:      []gorp.Migration{m1, m2},
					Instrumentation: ins,
				}))
				applied := logs.FilterMessage("already applied")
				Expect(applied.Len()).To(Equal(1))
				Expect(applied.All()[0].Level).To(Equal(zapcore.DebugLevel))
			})

			It("Should log migration failed on error", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				ins, logs := newObservedIns()
				migration := gorp.NewRawMigration(
					"bad_migration",
					func(context.Context, gorp.Tx) error {
						return errors.New("something broke")
					},
				)
				_, err := gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:              testDB,
					Migrations:      []gorp.Migration{migration},
					Instrumentation: ins,
				})
				Expect(err).To(HaveOccurred())
				failed := logs.FilterMessage("migration failed")
				Expect(failed.Len()).To(Equal(1))
				Expect(failed.All()[0].Level).To(Equal(zapcore.ErrorLevel))
				Expect(failed.All()[0].ContextMap()["migration"]).To(Equal("bad_migration"))
			})
		})

	})
})
