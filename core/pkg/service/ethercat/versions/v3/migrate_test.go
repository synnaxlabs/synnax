// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v3"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var (
	el3102 = v2.Slave{Inputs: []v2.Entry{
		{Name: "Value", Index: 0x6000, SubIndex: 0x11, DataType: "int16"},
		{Name: "Repeated", Index: 0x6010, SubIndex: 0x11, DataType: "int16"},
		{Name: "Repeated", Index: 0x6020, SubIndex: 0x11, DataType: "int16"},
	}}
	el2008 = v2.Slave{Outputs: []v2.Entry{
		{Name: "Output", Index: 0x7000, SubIndex: 1, DataType: "uint8"},
		{Name: "Repeated", Index: 0x7010, SubIndex: 1, DataType: "uint8"},
		{Name: "Repeated", Index: 0x7020, SubIndex: 1, DataType: "uint8"},
	}}
	readBase  = v2.BaseReadChannel{Key: "a", Device: "el3102", Channel: 12}
	writeBase = v2.BaseWriteChannel{
		Key:          "a",
		Device:       "el2008",
		CmdChannel:   3,
		StateChannel: 4,
	}
	none = task.Scale{Variant: task.NoneScale{}}
)

func findSlaves(
	_ context.Context,
	_ gorp.Tx,
	keys []string,
) (map[string]v2.Slave, error) {
	slaves := map[string]v2.Slave{"el3102": el3102, "el2008": el2008}
	return lo.PickByKeys(slaves, keys), nil
}

func failLookup(context.Context, gorp.Tx, []string) (map[string]v2.Slave, error) {
	return nil, errors.New("lookup failed")
}

func unresolvedRead(pdo string) v3.ReadChannel {
	return v3.ReadChannel{
		Variant: v3.UnresolvedReadChannel{BaseReadChannel: readBase, Pdo: pdo},
	}
}

func unresolvedWrite(pdo string) v3.WriteChannel {
	return v3.WriteChannel{
		Variant: v3.UnresolvedWriteChannel{BaseWriteChannel: writeBase, Pdo: pdo},
	}
}

func retrieve[C gorp.Entry[uuid.UUID]](
	ctx context.Context,
	db *gorp.DB,
	key uuid.UUID,
	migrations ...gorp.Migration,
) C {
	GinkgoHelper()
	table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, C]{
		DB:         db,
		Migrations: migrations,
	}))
	var cfg C
	Expect(table.NewRetrieve().
		Where(gorp.MatchKeys[uuid.UUID, C](key)).
		Entry(&cfg).
		Exec(ctx, db)).To(Succeed())
	return cfg
}

var _ = Describe("Migrate", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	Describe("ReadMigration", func() {
		It("Should keep every field of a stored v2 config and scale no entry", func(
			ctx SpecContext,
		) {
			var old v2.ReadConfig
			old.Key = uuid.New()
			old.SampleRate = 100
			old.StreamRate = 10
			old.Channels = []v2.ReadChannel{
				{Variant: v2.EntryReadChannel{
					BaseReadChannel: readBase,
					Index:           0x6000,
					SubIndex:        0x11,
					DataType:        "int16",
				}},
				{Variant: v2.UnresolvedReadChannel{
					BaseReadChannel: readBase,
					Pdo:             "Repeated",
				}},
			}
			Expect(gorp.WrapWriter[uuid.UUID, v2.ReadConfig](db).Set(ctx, old)).
				To(Succeed())
			Expect(
				retrieve[v3.ReadConfig](ctx, db, old.Key, v3.ReadMigration),
			).To(Equal(v3.ReadConfig{
				ReadConfig: old.ReadConfig,
				Channels: []v3.ReadChannel{
					{Variant: v3.EntryReadChannel{
						BaseReadChannel: readBase,
						Index:           0x6000,
						SubIndex:        0x11,
						DataType:        "int16",
						Scale:           none,
					}},
					unresolvedRead("Repeated"),
				},
			}))
		})

		It("Should lift a stored v1 config through every migration", func(
			ctx SpecContext,
		) {
			var old v1.ReadConfig
			old.Key = uuid.New()
			old.Channels = []v1.ReadChannel{{Variant: v1.AutomaticReadChannel{
				BaseReadChannel: readBase,
				Pdo:             "Value",
			}}}
			Expect(gorp.WrapWriter[uuid.UUID, v1.ReadConfig](db).Set(ctx, old)).
				To(Succeed())
			migrations := []gorp.Migration{
				v2.NewReadMigration(findSlaves),
				v3.ReadMigration,
			}
			Expect(
				retrieve[v3.ReadConfig](ctx, db, old.Key, migrations...).Channels,
			).To(Equal(
				[]v3.ReadChannel{{Variant: v3.EntryReadChannel{
					BaseReadChannel: readBase,
					Index:           0x6000,
					SubIndex:        0x11,
					DataType:        "int16",
					Scale:           none,
				}}},
			))
		})
	})

	Describe("ResolveReadConfig", func() {
		It("Should address a unique input entry name with no scale", func(
			ctx SpecContext,
		) {
			cfg := v3.ReadConfig{Channels: []v3.ReadChannel{unresolvedRead("Value")}}
			Expect(v3.ResolveReadConfig(ctx, nil, findSlaves, &cfg)).To(Succeed())
			Expect(cfg.Channels[0].Variant).To(Equal(v3.EntryReadChannel{
				BaseReadChannel: readBase,
				Index:           0x6000,
				SubIndex:        0x11,
				DataType:        "int16",
				Scale:           none,
			}))
		})

		It("Should leave a name at several addresses unresolved", func(
			ctx SpecContext,
		) {
			cfg := v3.ReadConfig{Channels: []v3.ReadChannel{unresolvedRead("Repeated")}}
			Expect(v3.ResolveReadConfig(ctx, nil, findSlaves, &cfg)).To(Succeed())
			Expect(cfg.Channels[0]).To(Equal(unresolvedRead("Repeated")))
		})

		It("Should return the error of the device lookup", func(ctx SpecContext) {
			cfg := v3.ReadConfig{Channels: []v3.ReadChannel{unresolvedRead("Value")}}
			Expect(v3.ResolveReadConfig(ctx, nil, failLookup, &cfg)).
				To(MatchError("lookup failed"))
		})
	})

	Describe("WriteMigration", func() {
		It("Should keep every field of a stored v2 config and stop each entry at 0",
			func(ctx SpecContext) {
				var old v2.WriteConfig
				old.Key = uuid.New()
				old.StateRate = 50
				old.ExecutionRate = 500
				old.Channels = []v2.WriteChannel{
					{Variant: v2.EntryWriteChannel{
						BaseWriteChannel: writeBase,
						Index:            0x7000,
						SubIndex:         1,
						DataType:         "uint8",
					}},
					{Variant: v2.UnresolvedWriteChannel{
						BaseWriteChannel: writeBase,
						Pdo:              "Repeated",
					}},
				}
				Expect(gorp.WrapWriter[uuid.UUID, v2.WriteConfig](db).Set(ctx, old)).
					To(Succeed())
				Expect(
					retrieve[v3.WriteConfig](ctx, db, old.Key, v3.WriteMigration),
				).To(Equal(v3.WriteConfig{
					PersistConfig: old.PersistConfig,
					StateRate:     50,
					ExecutionRate: 500,
					Channels: []v3.WriteChannel{
						{Variant: v3.EntryWriteChannel{
							BaseWriteChannel: writeBase,
							Index:            0x7000,
							SubIndex:         1,
							DataType:         "uint8",
						}},
						unresolvedWrite("Repeated"),
					},
				}))
			},
		)

		It("Should lift a stored v1 config through every migration", func(
			ctx SpecContext,
		) {
			var old v1.WriteConfig
			old.Key = uuid.New()
			old.Channels = []v1.WriteChannel{{Variant: v1.AutomaticWriteChannel{
				BaseWriteChannel: writeBase,
				Pdo:              "Output",
			}}}
			Expect(gorp.WrapWriter[uuid.UUID, v1.WriteConfig](db).Set(ctx, old)).
				To(Succeed())
			migrations := []gorp.Migration{
				v2.NewWriteMigration(findSlaves),
				v3.WriteMigration,
			}
			Expect(
				retrieve[v3.WriteConfig](ctx, db, old.Key, migrations...).Channels,
			).To(Equal(
				[]v3.WriteChannel{{Variant: v3.EntryWriteChannel{
					BaseWriteChannel: writeBase,
					Index:            0x7000,
					SubIndex:         1,
					DataType:         "uint8",
				}}},
			))
		})
	})

	Describe("ResolveWriteConfig", func() {
		It("Should address a unique output entry name", func(ctx SpecContext) {
			cfg := v3.WriteConfig{
				Channels: []v3.WriteChannel{unresolvedWrite("Output")},
			}
			Expect(v3.ResolveWriteConfig(ctx, nil, findSlaves, &cfg)).To(Succeed())
			Expect(cfg.Channels[0].Variant).To(Equal(v3.EntryWriteChannel{
				BaseWriteChannel: writeBase,
				Index:            0x7000,
				SubIndex:         1,
				DataType:         "uint8",
			}))
		})

		It("Should leave a name at several addresses unresolved", func(
			ctx SpecContext,
		) {
			cfg := v3.WriteConfig{
				Channels: []v3.WriteChannel{unresolvedWrite("Repeated")},
			}
			Expect(v3.ResolveWriteConfig(ctx, nil, findSlaves, &cfg)).To(Succeed())
			Expect(cfg.Channels[0]).To(Equal(unresolvedWrite("Repeated")))
		})

		It("Should return the error of the device lookup", func(ctx SpecContext) {
			cfg := v3.WriteConfig{
				Channels: []v3.WriteChannel{unresolvedWrite("Output")},
			}
			Expect(v3.ResolveWriteConfig(ctx, nil, failLookup, &cfg)).
				To(MatchError("lookup failed"))
		})
	})
})
