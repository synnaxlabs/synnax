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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	el2008 = v2.Slave{Outputs: []v2.Entry{
		{Name: "Output", Index: 0x7000, SubIndex: 1, DataType: "uint8"},
		{Name: "Repeated", Index: 0x7010, SubIndex: 1, DataType: "uint8"},
		{Name: "Repeated", Index: 0x7020, SubIndex: 1, DataType: "uint8"},
	}}
	base = v2.BaseWriteChannel{
		Key:          "a",
		Device:       "el2008",
		CmdChannel:   3,
		StateChannel: 4,
	}
)

func findSlaves(
	_ context.Context,
	_ gorp.Tx,
	keys []string,
) (map[string]v2.Slave, error) {
	return lo.PickByKeys(map[string]v2.Slave{"el2008": el2008}, keys), nil
}

func failLookup(context.Context, gorp.Tx, []string) (map[string]v2.Slave, error) {
	return nil, errors.New("lookup failed")
}

func unresolved(pdo string) v3.WriteChannel {
	return v3.WriteChannel{
		Variant: v3.UnresolvedWriteChannel{BaseWriteChannel: base, Pdo: pdo},
	}
}

func retrieve(
	ctx context.Context,
	db *gorp.DB,
	key uuid.UUID,
	migrations ...gorp.Migration,
) v3.WriteConfig {
	GinkgoHelper()
	table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v3.WriteConfig]{
		DB:         db,
		Migrations: migrations,
	}))
	var cfg v3.WriteConfig
	Expect(table.NewRetrieve().
		Where(gorp.MatchKeys[uuid.UUID, v3.WriteConfig](key)).
		Entry(&cfg).
		Exec(ctx, db)).To(Succeed())
	return cfg
}

var _ = Describe("Migrate", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	Describe("WriteMigration", func() {
		It("Should keep every field of a stored v2 config and stop each entry at 0",
			func(ctx SpecContext) {
				var old v2.WriteConfig
				old.Key = uuid.New()
				old.StateRate = 50
				old.ExecutionRate = 500
				old.Channels = []v2.WriteChannel{
					{Variant: v2.EntryWriteChannel{
						BaseWriteChannel: base,
						Index:            0x7000,
						SubIndex:         1,
						DataType:         "uint8",
					}},
					{Variant: v2.UnresolvedWriteChannel{
						BaseWriteChannel: base,
						Pdo:              "Repeated",
					}},
				}
				Expect(gorp.WrapWriter[uuid.UUID, v2.WriteConfig](db).Set(ctx, old)).
					To(Succeed())
				Expect(
					retrieve(ctx, db, old.Key, v3.WriteMigration),
				).To(Equal(v3.WriteConfig{
					PersistConfig: old.PersistConfig,
					StateRate:     50,
					ExecutionRate: 500,
					Channels: []v3.WriteChannel{
						{Variant: v3.EntryWriteChannel{
							BaseWriteChannel: base,
							Index:            0x7000,
							SubIndex:         1,
							DataType:         "uint8",
						}},
						unresolved("Repeated"),
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
				BaseWriteChannel: base,
				Pdo:              "Output",
			}}}
			Expect(gorp.WrapWriter[uuid.UUID, v1.WriteConfig](db).Set(ctx, old)).
				To(Succeed())
			migrations := []gorp.Migration{
				v2.NewWriteMigration(findSlaves),
				v3.WriteMigration,
			}
			Expect(retrieve(ctx, db, old.Key, migrations...).Channels).To(Equal(
				[]v3.WriteChannel{{Variant: v3.EntryWriteChannel{
					BaseWriteChannel: base,
					Index:            0x7000,
					SubIndex:         1,
					DataType:         "uint8",
				}}},
			))
		})
	})

	Describe("ResolveWriteConfig", func() {
		It("Should address a unique output entry name", func(ctx SpecContext) {
			cfg := v3.WriteConfig{Channels: []v3.WriteChannel{unresolved("Output")}}
			Expect(v3.ResolveWriteConfig(ctx, nil, findSlaves, &cfg)).To(Succeed())
			Expect(cfg.Channels[0].Variant).To(Equal(v3.EntryWriteChannel{
				BaseWriteChannel: base,
				Index:            0x7000,
				SubIndex:         1,
				DataType:         "uint8",
			}))
		})

		It("Should leave a name at several addresses unresolved", func(
			ctx SpecContext,
		) {
			cfg := v3.WriteConfig{Channels: []v3.WriteChannel{unresolved("Repeated")}}
			Expect(v3.ResolveWriteConfig(ctx, nil, findSlaves, &cfg)).To(Succeed())
			Expect(cfg.Channels[0]).To(Equal(unresolved("Repeated")))
		})

		It("Should return the error of the device lookup", func(ctx SpecContext) {
			cfg := v3.WriteConfig{Channels: []v3.WriteChannel{unresolved("Output")}}
			Expect(v3.ResolveWriteConfig(ctx, nil, failLookup, &cfg)).
				To(MatchError("lookup failed"))
		})
	})
})
