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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
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

func (jsonEntryCodec) Marshal(_ context.Context, e jsonEntry) ([]byte, error) {
	return json.Marshal(e)
}

func (jsonEntryCodec) Unmarshal(_ context.Context, data []byte) (jsonEntry, error) {
	var e jsonEntry
	err := json.Unmarshal(data, &e)
	return e, err
}

var _ gorp.Codec[jsonEntry] = jsonEntryCodec{}

type failMarshalCodec struct{}

func (failMarshalCodec) Marshal(_ context.Context, _ jsonEntry) ([]byte, error) {
	return nil, errors.New("marshal failed")
}

func (failMarshalCodec) Unmarshal(_ context.Context, data []byte) (jsonEntry, error) {
	var e jsonEntry
	err := json.Unmarshal(data, &e)
	return e, err
}

var _ gorp.Codec[jsonEntry] = failMarshalCodec{}

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

func (structpbCodec) Marshal(_ context.Context, e mapEntry) ([]byte, error) {
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

func (structpbCodec) Unmarshal(_ context.Context, data []byte) (mapEntry, error) {
	pb := &structpb.Struct{}
	if err := proto.Unmarshal(data, pb); err != nil {
		return mapEntry{}, err
	}
	m := pb.AsMap()
	var e mapEntry
	e.ID = int32(m["id"].(float64))
	e.Name = m["name"].(string)
	if d, ok := m["data"]; ok && d != nil {
		e.Data = d.(map[string]any)
	}
	return e, nil
}

var _ gorp.Codec[mapEntry] = structpbCodec{}

var _ = Describe("Gorp", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})
	Describe("Codec", func() {
		It("Should use Codec.Marshal when a codec is provided to the table", func() {
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

		It("Should fall back to DB codec when no table codec is provided", func() {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entry](testDB)
			Expect(w.Set(ctx, entry{ID: 1, Data: "hello"})).To(Succeed())
			r := gorp.WrapReader[int32, entry](testDB)
			result := MustSucceed(r.Get(ctx, 1))
			Expect(result.ID).To(Equal(int32(1)))
			Expect(result.Data).To(Equal("hello"))
		})

		It("Should use Codec.Unmarshal in Iterator.Value via table reader", func() {
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
			It("Should transform entries from one schema to another", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
				Expect(w.Set(ctx, entryV1{ID: 2, Data: "two"})).To(Succeed())
				migration := gorp.NewTypedMigration[entryV1, entryV1](
					"add_suffix",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_migrated"}, nil
					},
					nil,
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("one_migrated"))
				Expect(MustSucceed(r.Get(ctx, 2)).Data).To(Equal("two_migrated"))
			})

			It("Should call PostMigrateFunc after auto migration", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
				migration := gorp.NewTypedMigration[entryV1, entryV1](
					"post_transform",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return old, nil
					},
					func(_ context.Context, new *entryV1, old entryV1) error {
						new.Data = "post:" + old.Data
						return nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("post:one"))
			})

			It("Should handle cross-type migration with manually seeded data", func() {
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
				migration := gorp.NewTypedMigration[entryV1, entryV2](
					"v1_to_v2",
					func(_ context.Context, old entryV1) (entryV2, error) {
						return entryV2{
							ID:          old.ID,
							Data:        old.Data,
							Description: "migrated:" + old.Data,
						}, nil
					},
					nil,
				)
				MustSucceed(gorp.OpenTable[int32, entryV2](ctx, gorp.TableConfig[entryV2]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV2](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Description).To(Equal("migrated:one"))
				Expect(MustSucceed(r.Get(ctx, 2)).Description).To(Equal("migrated:two"))
			})
		})

		Describe("RawMigration", func() {
			It("Should provide a working gorp.Tx for read/write", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "raw"})).To(Succeed())
				migration := gorp.NewRawMigration(
					"raw_transform",
					func(ctx context.Context, tx gorp.Tx) error {
						r := gorp.WrapReader[int32, entryV1](tx)
						e := MustSucceed(r.Get(ctx, 1))
						e.Data = "raw_migrated"
						w := gorp.WrapWriter[int32, entryV1](tx)
						return w.Set(ctx, e)
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("raw_migrated"))
			})
		})

		Describe("Version tracking", func() {
			It("Should store version as uint16 big-endian", func() {
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
				Expect(b).To(HaveLen(2))
				Expect(stdbinary.BigEndian.Uint16(b)).To(Equal(uint16(1)))
			})

			It("Should skip already-completed migrations on re-run", func() {
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

			It("Should only run new migrations after partial completion", func() {
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
			It("Should chain two TypedMigrations sequentially", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "chain"})).To(Succeed())
				m1 := gorp.NewTypedMigration[entryV1, entryV1](
					"add_suffix",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_v2"}, nil
					},
					nil,
				)
				m2 := gorp.NewTypedMigration[entryV1, entryV1](
					"add_suffix_2",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_v3"}, nil
					},
					nil,
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("chain_v2_v3"))
			})

			It("Should chain a TypedMigration with a RawMigration", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "mixed"})).To(Succeed())
				m1 := gorp.NewTypedMigration[entryV1, entryV1](
					"typed_transform",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_typed"}, nil
					},
					nil,
				)
				m2 := gorp.NewRawMigration("raw_update", func(ctx context.Context, tx gorp.Tx) error {
					r := gorp.WrapReader[int32, entryV1](tx)
					e := MustSucceed(r.Get(ctx, 1))
					e.Data = e.Data + "_raw"
					w := gorp.WrapWriter[int32, entryV1](tx)
					return w.Set(ctx, e)
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("mixed_typed_raw"))
			})
		})

		Describe("Error handling", func() {
			It("Should not commit when a migration fails", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
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
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("original"))
			})
		})

		Describe("CodecTransitionMigration", func() {
			It("Should re-encode entries from default codec to custom codec", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB)
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

			It("Should produce bytes in the target codec format", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB)
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

			It("Should be a no-op on second run", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB)
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

			It("Should handle empty DB", func() {
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

			It("Should migrate many entries correctly", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB)
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

			It("Should return an error when source data cannot be decoded", func() {
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

			It("Should return an error when target codec fails to marshal", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB)
				Expect(w.Set(ctx, jsonEntry{ID: 1, Data: "fail"})).To(Succeed())
				Expect(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
					DB:    testDB,
					Codec: failMarshalCodec{},
					Migrations: []gorp.Migration{
						gorp.NewCodecTransition[int32, jsonEntry]("to_failing", failMarshalCodec{}),
					},
				})).Error().To(MatchError(ContainSubstring("marshal failed")))
			})

			It("Should work in a migration chain with a preceding RawMigration", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB)
				Expect(w.Set(ctx, jsonEntry{ID: 1, Data: "chain"})).To(Succeed())
				m1 := gorp.NewRawMigration("raw_update", func(ctx context.Context, tx gorp.Tx) error {
					r := gorp.WrapReader[int32, jsonEntry](tx)
					e := MustSucceed(r.Get(ctx, 1))
					e.Data = e.Data + "_raw"
					w := gorp.WrapWriter[int32, jsonEntry](tx)
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

			It("Should not commit when migration fails", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, jsonEntry](testDB)
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
				r := gorp.WrapReader[int32, jsonEntry](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("original"))
			})

			It("Should re-encode MsgpackEncodedJSON fields through structpb", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, mapEntry](testDB)
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
			It("Should run key re-encoding only when no migrations are provided", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "no_migration"})).To(Succeed())
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB: testDB,
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("no_migration"))
			})
		})

		Describe("Idempotent key migration", func() {
			It("Should run key re-encoding even when versioned migrations are at latest", func() {
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
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 5, Data: "post_migration"})).To(Succeed())
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 5)).Data).To(Equal("post_migration"))
			})
		})
	})
})
