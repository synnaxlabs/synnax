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
	el3102 = v2.Slave{Inputs: []v2.Entry{
		{Name: "Value", Index: 0x6000, SubIndex: 0x11, DataType: "int16"},
		{Name: "Repeated", Index: 0x6010, SubIndex: 0x11, DataType: "int16"},
		{Name: "Repeated", Index: 0x6020, SubIndex: 0x11, DataType: "int16"},
	}}
	base = v2.BaseReadChannel{Key: "a", Device: "el3102", Channel: 12}
	none = v3.Scale{Variant: v3.NoneScale{}}
)

func findSlaves(
	_ context.Context,
	_ gorp.Tx,
	keys []string,
) (map[string]v2.Slave, error) {
	return lo.PickByKeys(map[string]v2.Slave{"el3102": el3102}, keys), nil
}

func failLookup(context.Context, gorp.Tx, []string) (map[string]v2.Slave, error) {
	return nil, errors.New("lookup failed")
}

func unresolved(pdo string) v3.ReadChannel {
	return v3.ReadChannel{
		Variant: v3.UnresolvedReadChannel{BaseReadChannel: base, Pdo: pdo},
	}
}

func retrieve(
	ctx context.Context,
	db *gorp.DB,
	key uuid.UUID,
	migrations ...gorp.Migration,
) v3.ReadConfig {
	GinkgoHelper()
	table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v3.ReadConfig]{
		DB:         db,
		Migrations: migrations,
	}))
	var cfg v3.ReadConfig
	Expect(table.NewRetrieve().
		Where(gorp.MatchKeys[uuid.UUID, v3.ReadConfig](key)).
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
					BaseReadChannel: base,
					Index:           0x6000,
					SubIndex:        0x11,
					DataType:        "int16",
				}},
				{Variant: v2.UnresolvedReadChannel{
					BaseReadChannel: base,
					Pdo:             "Repeated",
				}},
			}
			Expect(gorp.WrapWriter[uuid.UUID, v2.ReadConfig](db).Set(ctx, old)).
				To(Succeed())
			Expect(
				retrieve(ctx, db, old.Key, v3.ReadMigration),
			).To(Equal(v3.ReadConfig{
				ReadConfig: old.ReadConfig,
				Channels: []v3.ReadChannel{
					{Variant: v3.EntryReadChannel{
						BaseReadChannel: base,
						Index:           0x6000,
						SubIndex:        0x11,
						DataType:        "int16",
						Scale:           none,
					}},
					unresolved("Repeated"),
				},
			}))
		})

		It("Should lift a stored v1 config through every migration", func(
			ctx SpecContext,
		) {
			var old v1.ReadConfig
			old.Key = uuid.New()
			old.Channels = []v1.ReadChannel{{Variant: v1.AutomaticReadChannel{
				BaseReadChannel: base,
				Pdo:             "Value",
			}}}
			Expect(gorp.WrapWriter[uuid.UUID, v1.ReadConfig](db).Set(ctx, old)).
				To(Succeed())
			migrations := []gorp.Migration{
				v2.NewReadMigration(findSlaves),
				v3.ReadMigration,
			}
			Expect(retrieve(ctx, db, old.Key, migrations...).Channels).To(Equal(
				[]v3.ReadChannel{{Variant: v3.EntryReadChannel{
					BaseReadChannel: base,
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
			cfg := v3.ReadConfig{Channels: []v3.ReadChannel{unresolved("Value")}}
			Expect(v3.ResolveReadConfig(ctx, nil, findSlaves, &cfg)).To(Succeed())
			Expect(cfg.Channels[0].Variant).To(Equal(v3.EntryReadChannel{
				BaseReadChannel: base,
				Index:           0x6000,
				SubIndex:        0x11,
				DataType:        "int16",
				Scale:           none,
			}))
		})

		It("Should leave a name at several addresses unresolved", func(
			ctx SpecContext,
		) {
			cfg := v3.ReadConfig{Channels: []v3.ReadChannel{unresolved("Repeated")}}
			Expect(v3.ResolveReadConfig(ctx, nil, findSlaves, &cfg)).To(Succeed())
			Expect(cfg.Channels[0]).To(Equal(unresolved("Repeated")))
		})

		It("Should return the error of the device lookup", func(ctx SpecContext) {
			cfg := v3.ReadConfig{Channels: []v3.ReadChannel{unresolved("Value")}}
			Expect(v3.ResolveReadConfig(ctx, nil, failLookup, &cfg)).
				To(MatchError("lookup failed"))
		})
	})
})
