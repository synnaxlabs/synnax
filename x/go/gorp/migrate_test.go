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
					Migrations: []gorp.Migration{migration},
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
					Migrations: []gorp.Migration{migration},
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
					Migrations: []gorp.Migration{migration},
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
					Migrations: []gorp.Migration{migration},
				}))
				versionKey := []byte("__gorp_migration__//entryV1")
				b, closer := MustSucceed2(testDB.Get(ctx, versionKey))
				Expect(closer.Close()).To(Succeed())
				Expect(string(b)).To(Equal("noop\nnormalize_keys"))
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
				m1 := gorp.NewMigration("first", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					executed = append(executed, "first")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1},
				}))
				Expect(executed).To(Equal([]string{"first"}))
				m2 := gorp.NewMigration("second", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
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
					Migrations: []gorp.Migration{m1, m2},
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
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, gorp.WithDependencies(m2, "typed_transform")},
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
					Migrations: []gorp.Migration{migration},
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

		Describe("WithDependencies", func() {
			It("Should order migrations respecting declared dependencies", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var order []string
				m1 := gorp.NewMigration("alpha", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "alpha")
					return nil
				})
				m2 := gorp.WithDependencies(
					gorp.NewMigration("beta", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
						order = append(order, "beta")
						return nil
					}),
					"alpha",
				)
				m3 := gorp.WithDependencies(
					gorp.NewMigration("gamma", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
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
				m1 := gorp.NewMigration("first", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "first")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1},
				}))
				Expect(order).To(Equal([]string{"first"}))
				m2 := gorp.WithDependencies(
					gorp.NewMigration("second", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
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
					gorp.NewMigration("a", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil }),
					"b",
				)
				m2 := gorp.WithDependencies(
					gorp.NewMigration("b", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil }),
					"a",
				)
				Expect(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				})).Error().To(MatchError(ContainSubstring("cyclic dependency")))
			})

			It("Should return error for missing dependency", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				m1 := gorp.WithDependencies(
					gorp.NewMigration("orphan", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil }),
					"nonexistent",
				)
				Expect(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1},
				})).Error().To(MatchError(ContainSubstring("nonexistent")))
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
					Migrations: []gorp.Migration{m1, m2},
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
				m2 := gorp.WithDependencies(
					gorp.NewMigration("has_dep", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
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
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "original"})).To(Succeed())
				type LookupService struct {
					Suffix string
				}
				depCtx := gorp.WithMigrationDep[*LookupService](ctx, &LookupService{Suffix: "_enriched"})
				migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"enrich",
					func(ctx context.Context, old entryV1) (entryV1, error) {
						svc, err := gorp.MigrationDep[*LookupService](ctx)
						if err != nil {
							return entryV1{}, err
						}
						return entryV1{ID: old.ID, Data: old.Data + svc.Suffix}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](depCtx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("original_enriched"))
			})

			It("Should support multiple dependencies of different types", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "base"})).To(Succeed())
				type PrefixService struct {
					Prefix string
				}
				type SuffixService struct {
					Suffix string
				}
				depCtx := gorp.WithMigrationDep[*PrefixService](ctx, &PrefixService{Prefix: "pre_"})
				depCtx = gorp.WithMigrationDep[*SuffixService](depCtx, &SuffixService{Suffix: "_post"})
				migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"wrap",

					func(ctx context.Context, old entryV1) (entryV1, error) {
						pre, err := gorp.MigrationDep[*PrefixService](ctx)
						if err != nil {
							return entryV1{}, err
						}
						suf, err := gorp.MigrationDep[*SuffixService](ctx)
						if err != nil {
							return entryV1{}, err
						}
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
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("pre_base_post"))
			})

			It("Should return an error when a required dependency is missing", func(ctx SpecContext) {
				type MissingService struct{}
				Expect(gorp.MigrationDep[*MissingService](ctx)).Error().To(MatchError(query.ErrNotFound))
			})

			It("Should work with RawMigration", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "raw"})).To(Succeed())
				type Renamer struct {
					NewData string
				}
				depCtx := gorp.WithMigrationDep[*Renamer](ctx, &Renamer{NewData: "renamed"})
				migration := gorp.NewMigration(
					"raw_with_dep",
					func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
						renamer, err := gorp.MigrationDep[*Renamer](ctx)
						if err != nil {
							return err
						}
						r := gorp.WrapReader[int32, entryV1](tx)
						e := MustSucceed(r.Get(ctx, 1))
						e.Data = renamer.NewData
						w := gorp.WrapWriter[int32, entryV1](tx)
						return w.Set(ctx, e)
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](depCtx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("renamed"))
			})

			It("Should work with interface types", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "iface"})).To(Succeed())
				depCtx := gorp.WithMigrationDep[migrationDepProvider](
					ctx, migrationDepProviderImpl("_from_iface"),
				)
				migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"iface_dep",

					func(ctx context.Context, old entryV1) (entryV1, error) {
						dp, err := gorp.MigrationDep[migrationDepProvider](ctx)
						if err != nil {
							return entryV1{}, err
						}
						return entryV1{ID: old.ID, Data: old.Data + dp.GetSuffix()}, nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](depCtx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("iface_from_iface"))
			})
		})

		Describe("EntryCounter", func() {
			It("Should track entries processed by TypedMigration", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				for i := int32(0); i < 5; i++ {
					Expect(w.Set(ctx, entryV1{ID: i, Data: "x"})).To(Succeed())
				}
				migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"count_test",

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
					Migrations: []gorp.Migration{migration},
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
					Migrations: []gorp.Migration{migration},
				})).Error().To(MatchError(ContainSubstring("decode")))
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
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "x"})).To(Succeed())
				ins, logs := newObservedIns()
				migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
					"test_migration",

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
				// 1 built-in migration + 1 user migration
				Expect(starting.All()[0].ContextMap()["pending"]).To(BeNumerically("==", 2))

				complete := logs.FilterMessage("migration complete")
				Expect(complete.Len()).To(Equal(2))
				names := make([]string, complete.Len())
				for i, entry := range complete.All() {
					names[i] = entry.ContextMap()["migration"].(string)
				}
				Expect(names).To(ContainElement("test_migration"))

				done := logs.FilterMessage("migrations complete")
				Expect(done.Len()).To(Equal(1))
				Expect(done.All()[0].ContextMap()["migrations"]).To(BeNumerically("==", 2))
			})

			It("Should not log when all migrations are already applied", func(ctx SpecContext) {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				migration := gorp.NewMigration(
					"noop",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
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
				m1 := gorp.NewMigration("first", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil })
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1},
				}))
				ins, logs := newObservedIns()
				m2 := gorp.NewMigration("second", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil })
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
				migration := gorp.NewMigration(
					"bad_migration",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
						return errors.New("something broke")
					},
				)
				_, err := gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:              testDB,
					Migrations:      []gorp.Migration{migration},
					Instrumentation: ins,
				})
				Expect(err).To(MatchError(ContainSubstring("bad")))
				failed := logs.FilterMessage("migration failed")
				Expect(failed.Len()).To(Equal(1))
				Expect(failed.All()[0].Level).To(Equal(zapcore.ErrorLevel))
				Expect(failed.All()[0].ContextMap()["migration"]).To(Equal("bad_migration"))
			})
		})

	})
})
