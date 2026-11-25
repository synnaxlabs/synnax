// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"context"
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v4"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var (
	slaves = map[string]v2.Slave{
		"in": {Inputs: []v2.Entry{
			{Name: "Value", Index: 0x6000, SubIndex: 0x11, DataType: "int16"},
			{Name: "Padding", Index: 0x6010, SubIndex: 0x11},
		}},
		"out": {Outputs: []v2.Entry{
			{Name: "Output", Index: 0x7000, SubIndex: 1, DataType: "uint8"},
			{Name: "Padding", Index: 0x7010, SubIndex: 1},
		}},
	}
	readBase  = v1.BaseReadChannel{Key: "a", Device: "in", Channel: 12}
	writeBase = v1.BaseWriteChannel{Key: "a", Device: "out", CmdChannel: 3}
	none      = task.Scale{Variant: task.NoneScale{}}
)

func findSlaves(
	_ context.Context,
	_ gorp.Snapshot,
	keys []string,
) (map[string]v2.Slave, error) {
	return lo.PickByKeys(slaves, keys), nil
}

var _ = Describe("Migrate", func() {
	DescribeTable("MigrateReadChannel",
		func(ctx SpecContext, old v3.ReadChannel, expected v4.ReadChannel) {
			Expect(v4.MigrateReadChannel(ctx, old)).To(Equal(expected))
		},
		Entry("an empty data type as absent",
			v3.ReadChannel{Variant: v3.EntryReadChannel{
				BaseReadChannel: readBase, Index: 0x6000, Scale: none,
			}},
			v4.ReadChannel{Variant: v4.EntryReadChannel{
				BaseReadChannel: readBase, Index: 0x6000, Scale: none,
			}},
		),
		Entry("a data type as itself",
			v3.ReadChannel{Variant: v3.EntryReadChannel{
				BaseReadChannel: readBase, DataType: "int16", Scale: none,
			}},
			v4.ReadChannel{Variant: v4.EntryReadChannel{
				BaseReadChannel: readBase, DataType: new(telem.Int16T), Scale: none,
			}},
		),
		Entry("an unresolved channel unchanged",
			v3.ReadChannel{Variant: v3.UnresolvedReadChannel{
				BaseReadChannel: readBase, Pdo: "Value",
			}},
			v4.ReadChannel{Variant: v4.UnresolvedReadChannel{
				BaseReadChannel: readBase, Pdo: "Value",
			}},
		),
	)

	DescribeTable("MigrateWriteChannel",
		func(ctx SpecContext, old v3.WriteChannel, expected v4.WriteChannel) {
			Expect(v4.MigrateWriteChannel(ctx, old)).To(Equal(expected))
		},
		Entry("an empty data type as absent",
			v3.WriteChannel{Variant: v3.EntryWriteChannel{
				BaseWriteChannel: writeBase, Index: 0x7000, StopValue: 1,
			}},
			v4.WriteChannel{Variant: v4.EntryWriteChannel{
				BaseWriteChannel: writeBase, Index: 0x7000, StopValue: 1,
			}},
		),
		Entry("a data type as itself",
			v3.WriteChannel{Variant: v3.EntryWriteChannel{
				BaseWriteChannel: writeBase, DataType: "uint8",
			}},
			v4.WriteChannel{Variant: v4.EntryWriteChannel{
				BaseWriteChannel: writeBase, DataType: new(telem.Uint8T),
			}},
		),
		Entry("an unresolved channel unchanged",
			v3.WriteChannel{Variant: v3.UnresolvedWriteChannel{
				BaseWriteChannel: writeBase, Pdo: "Output",
			}},
			v4.WriteChannel{Variant: v4.UnresolvedWriteChannel{
				BaseWriteChannel: writeBase, Pdo: "Output",
			}},
		),
	)

	It("Should lift a stored read config with an empty data type", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		key := uuid.New()
		old := v3.ReadConfig{
			Key: key,
			Channels: []v3.ReadChannel{{Variant: v3.EntryReadChannel{
				BaseReadChannel: readBase, Index: 0x6000, Scale: none,
			}}},
		}
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v3.ReadConfig]{DB: db}))
		Expect(gorp.NewCreate[uuid.UUID, v3.ReadConfig]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v4.ReadConfig]{
			DB: db, Migrations: []gorp.Migration{v4.ReadMigration},
		}))
		var got v4.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v4.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v4.ReadConfig](key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		entry, ok := got.Channels[0].Variant.(v4.EntryReadChannel)
		Expect(ok).To(BeTrue())
		Expect(entry.DataType).To(BeNil())
	})

	DescribeTable("ResolveReadConfig",
		func(ctx SpecContext, pdo string, expected *telem.DataType) {
			cfg := v4.ReadConfig{Channels: []v4.ReadChannel{{
				Variant: v4.UnresolvedReadChannel{BaseReadChannel: readBase, Pdo: pdo},
			}}}
			Expect(v4.ResolveReadConfig(ctx, nil, findSlaves, &cfg)).To(Succeed())
			entry, ok := cfg.Channels[0].Variant.(v4.EntryReadChannel)
			Expect(ok).To(BeTrue())
			Expect(entry.DataType).To(Equal(expected))
		},
		Entry("an entry with a data type", "Value", new(telem.Int16T)),
		Entry("an entry with no data type as absent", "Padding", nil),
	)

	DescribeTable("ResolveWriteConfig",
		func(ctx SpecContext, pdo string, expected *telem.DataType) {
			cfg := v4.WriteConfig{Channels: []v4.WriteChannel{{
				Variant: v4.UnresolvedWriteChannel{BaseWriteChannel: writeBase, Pdo: pdo},
			}}}
			Expect(v4.ResolveWriteConfig(ctx, nil, findSlaves, &cfg)).To(Succeed())
			entry, ok := cfg.Channels[0].Variant.(v4.EntryWriteChannel)
			Expect(ok).To(BeTrue())
			Expect(entry.DataType).To(Equal(expected))
		},
		Entry("an entry with a data type", "Output", new(telem.Uint8T)),
		Entry("an entry with no data type as absent", "Padding", nil),
	)

	DescribeTable("Should decode an empty entry data type as absent",
		func(data string, isRead bool) {
			if isRead {
				var ch v4.ReadChannel
				Expect(json.Unmarshal([]byte(data), &ch)).To(Succeed())
				entry, ok := ch.Variant.(v4.EntryReadChannel)
				Expect(ok).To(BeTrue())
				Expect(entry.DataType).To(BeNil())
				Expect(entry.Index).To(BeEquivalentTo(1))
				return
			}
			var ch v4.WriteChannel
			Expect(json.Unmarshal([]byte(data), &ch)).To(Succeed())
			entry, ok := ch.Variant.(v4.EntryWriteChannel)
			Expect(ok).To(BeTrue())
			Expect(entry.DataType).To(BeNil())
			Expect(entry.Index).To(BeEquivalentTo(1))
		},
		Entry("for a read channel", `{"type":"entry","index":1,"data_type":""}`, true),
		Entry("for a write channel", `{"type":"entry","index":1,"data_type":""}`, false),
	)
})
