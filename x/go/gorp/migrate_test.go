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
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/graph"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
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

var _ encoding.Codec = jsonEntryCodec{}

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

var _ encoding.Codec = failMarshalCodec{}

type migrationDepProvider interface {
	GetSuffix() string
}

type migrationDepProviderImpl string

func (p migrationDepProviderImpl) GetSuffix() string { return string(p) }

// mapEntry mimics production types like Schematic that have a MsgpackEncodedJSON field.
type mapEntry struct {
	ID   int32               `msgpack:"id"`
	Name string              `msgpack:"name"`
	Data msgpack.EncodedJSON `msgpack:"data"`
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

var _ encoding.Codec = structpbCodec{}

var _ = Describe("Gorp", func() {
	Describe("Codec", func() {
		It("Should use Codec.Marshal when a codec is provided to the table", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			tbl := MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
				DB: testDB,
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
			w := gorp.WrapWriter[int32, entry](testDB)
			Expect(w.Set(ctx, entry{ID: 1, Data: "hello"})).To(Succeed())
			r := gorp.WrapReader[int32, entry](testDB)
			result := MustSucceed(r.Get(ctx, 1))
			Expect(result.ID).To(Equal(int32(1)))
			Expect(result.Data).To(Equal("hello"))
		})

		It("Should use Codec.Unmarshal in Iterator.Value via table reader", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			tbl := MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
				DB: testDB,
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
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
				Expect(w.Set(ctx, entryV1{ID: 2, Data: "two"})).To(Succeed())
				migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"add_suffix",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_migrated"}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("one_migrated"))
				Expect(MustSucceed(r.Get(ctx, 2)).Data).To(Equal("two_migrated"))
			})

			It("Should call PostMigrateFunc after auto migration", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
				migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"post_transform",

					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: "post:" + old.Data}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("post:one"))
			})
		})

		Describe("RawMigration", func() {
			It("Should provide a working gorp.Tx for read/write", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "raw"})).To(Succeed())
				migration := gorp.NewMigration(
					"raw_transform",
					func(_ context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
						r := gorp.WrapReader[int32, entryV1](tx)
						e := MustSucceed(r.Get(ctx, 1))
						e.Data = "raw_migrated"
						w := gorp.WrapWriter[int32, entryV1](tx)
						return w.Set(ctx, e)
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("raw_migrated"))
			})
		})

		Describe("Version tracking", func() {
			It("Should store applied migration names", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				migration := gorp.NewMigration(
					"noop",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{migration},
				}))
				versionKey := []byte("__gorp_migration__//entryV1")
				b, closer := MustSucceed2(testDB.Get(ctx, versionKey))
				Expect(closer.Close()).To(Succeed())
				Expect(string(b)).To(Equal("[\"noop\",\"normalize_keys\"]"))
			})

			It("Should skip already-completed migrations on re-run", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				executionCount := 0
				migration := gorp.NewMigration(
					"counted",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
						executionCount++
						return nil
					},
				)
				cfg := gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{migration},
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
				m1 := gorp.NewMigration("first", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					executed = append(executed, "first")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{m1},
				}))
				Expect(executed).To(Equal([]string{"first"}))
				m2 := gorp.NewMigration("second", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					executed = append(executed, "second")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{m1, m2},
				}))
				Expect(executed).To(Equal([]string{"first", "second"}))
			})
		})

		Describe("Sequential execution / chaining", func() {
			It("Should chain two TypedMigrations sequentially", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "chain"})).To(Succeed())
				m1 := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"add_suffix",

					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_v2"}, nil
					},
				)
				m2 := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"add_suffix_2",

					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_v3"}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{m1, m2},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("chain_v2_v3"))
			})

			It("Should chain a TypedMigration with a RawMigration", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "mixed"})).To(Succeed())
				m1 := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"typed_transform",

					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_typed"}, nil
					},
				)
				m2 := gorp.NewMigration("raw_update", func(
					ctx context.Context,
					tx gorp.Tx,
					ins alamos.Instrumentation,
				) error {
					r := gorp.WrapReader[int32, entryV1](tx)
					e := MustSucceed(r.Get(ctx, 1))
					e.Data = e.Data + "_raw"
					w := gorp.WrapWriter[int32, entryV1](tx)
					return w.Set(ctx, e)
				}, "typed_transform")
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{m1, m2},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("mixed_typed_raw"))
			})
		})

		Describe("Error handling", func() {
			It("Should not commit when a migration fails", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "original"})).To(Succeed())
				migration := gorp.NewMigration(
					"failing",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
						return errors.New("migration error")
					},
				)
				Expect(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{migration},
				})).Error().To(MatchError(ContainSubstring("failing")))
				versionKey := []byte("__gorp_migration__//entryV1")
				Expect(testDB.Get(ctx, versionKey)).Error().To(MatchError(query.ErrNotFound))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("original"))
			})
		})

		Describe("Zero-migration case", func() {
			It("Should run key re-encoding only when no migrations are provided", func(ctx SpecContext) {
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
			It("Should run key re-encoding even when versioned migrations are at latest", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				migration := gorp.NewMigration(
					"noop",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
				)
				cfg := gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{migration},
				}
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 5, Data: "post_migration"})).To(Succeed())
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 5)).Data).To(Equal("post_migration"))
			})
		})

		Describe("WithDependencies", func() {
			It("Should order migrations respecting declared dependencies", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var order []string
				m1 := gorp.NewMigration("alpha", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "alpha")
					return nil
				})
				m2 := gorp.NewMigration("beta", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "beta")
					return nil
				}, "alpha")
				m3 := gorp.NewMigration("gamma", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "gamma")
					return nil
				}, "beta")
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{m3, m2, m1},
				}))
				Expect(order).To(Equal([]string{"alpha", "beta", "gamma"}))
			})

			It("Should treat already-applied dependencies as satisfied", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var order []string
				m1 := gorp.NewMigration("first", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "first")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{m1},
				}))
				Expect(order).To(Equal([]string{"first"}))
				m2 := gorp.NewMigration("second", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "second")
					return nil
				}, "first")
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{m1, m2},
				}))
				Expect(order).To(Equal([]string{"first", "second"}))
			})

			It("Should detect cyclic dependencies", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				m1 := gorp.NewMigration(
					"a",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
					"b",
				)
				m2 := gorp.NewMigration(
					"b",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
					"a",
				)
				Expect(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{m1, m2},
				})).Error().To(MatchError(graph.ErrCyclicDependency))
			})

			It("Should return error for missing dependency", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				m1 := gorp.NewMigration(
					"orphan",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
					"nonexistent",
				)
				Expect(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{m1},
				})).Error().To(MatchError(query.ErrNotFound))
			})

			It("Should work with migrations that do not declare dependencies", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var order []string
				m1 := gorp.NewMigration("plain_a", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "plain_a")
					return nil
				})
				m2 := gorp.NewMigration("plain_b", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "plain_b")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{m1, m2},
				}))
				Expect(order).To(Equal([]string{"plain_a", "plain_b"}))
			})

			It("Should preserve insertion order for migrations without dependencies "+
				"when mixed with dependency-declaring migrations", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var order []string
				m1 := gorp.NewMigration("no_dep", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "no_dep")
					return nil
				})
				m2 := gorp.NewMigration("has_dep", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "has_dep")
					return nil
				}, "no_dep")
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{m2, m1},
				}))
				Expect(order).To(Equal([]string{"no_dep", "has_dep"}))
			})
		})

		Describe("Error context", func() {
			It("Should include entry key in transform error", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 42, Data: "bad"})).To(Succeed())
				migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"fail_transform",

					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{}, errors.New("transform broke")
					},
				)
				Expect(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{migration},
				})).Error().To(MatchError(ContainSubstring("transform")))
			})

			It("Should include raw key in decode error", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				prefix := "__gorp__//entryV1"
				key := make([]byte, len(prefix)+4)
				copy(key, prefix)
				stdbinary.BigEndian.PutUint32(key[len(prefix):], 1)
				Expect(testDB.Set(ctx, key, []byte("not valid msgpack"))).To(Succeed())
				migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"fail_decode",
					func(ctx context.Context, old entryV1) (entryV1, error) {
						return old, nil
					},
				)
				Expect(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []migrate.Migration{migration},
				})).Error().To(MatchError(ContainSubstring("decode")))
			})
		})
	})
})
