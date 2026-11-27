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
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v3"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("MigrateReadConfig", func() {
	It("Should drop the empty data type of each binding", func(ctx SpecContext) {
		field := uuid.New()
		old := v2.ReadConfig{Entries: []v2.ReadEntry{{Variant: v2.PlainReadEntry{
			Topic: "plant/a",
			Messages: []icdv0.ReadMessage{{
				Message:  uuid.New(),
				Bindings: []icdv0.ReadBinding{{Field: field, Channel: 7}},
			}},
		}}}}
		cfg := MustSucceed(v3.MigrateReadConfig(ctx, old))
		plain, ok := cfg.Entries[0].Variant.(v3.PlainReadEntry)
		Expect(ok).To(BeTrue())
		Expect(plain.Topic).To(Equal("plant/a"))
		Expect(plain.Messages[0].Bindings).To(Equal([]icd.ReadBinding{
			{Field: field, Channel: 7},
		}))
	})

	It("Should keep the inline entries", func(ctx SpecContext) {
		inline := &v2.InlineRead{Entries: []v2.InlineEntry{{Topic: "plant/a"}}}
		cfg := MustSucceed(v3.MigrateReadConfig(ctx, v2.ReadConfig{Inline: inline}))
		Expect(cfg.Inline).To(Equal(inline))
	})
})

var _ = Describe("ReadMigration", func() {
	It("Should lift each stored read config", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(
			gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.ReadConfig]{DB: db}),
		)
		field := uuid.New()
		old := v2.ReadConfig{
			Key: uuid.New(),
			Entries: []v2.ReadEntry{{Variant: v2.PlainReadEntry{
				Topic: "plant/a",
				Messages: []icdv0.ReadMessage{{
					Message:  uuid.New(),
					Bindings: []icdv0.ReadBinding{{Field: field, Channel: 7}},
				}},
			}}},
		}
		Expect(gorp.NewCreate[uuid.UUID, v2.ReadConfig]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v3.ReadConfig]{
			DB: db, Migrations: []gorp.Migration{v3.ReadMigration},
		}))
		var got v3.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v3.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v3.ReadConfig](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		plain, ok := got.Entries[0].Variant.(v3.PlainReadEntry)
		Expect(ok).To(BeTrue())
		Expect(plain.Messages[0].Bindings).To(Equal([]icd.ReadBinding{
			{Field: field, Channel: 7},
		}))
	})
})

var _ = Describe("ResolveReadConfig", func() {
	It("Should add each inline entry back, read from an ICD v1 message", func(
		ctx SpecContext,
	) {
		channels := func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]icdv0.Channel, error) {
			return map[channel.Key]icdv0.Channel{
				7: {Name: "level", DataType: telem.Float64T},
			}, nil
		}
		cfg := v3.ReadConfig{Inline: &v3.InlineRead{Entries: []v3.InlineEntry{{
			At:    0,
			Topic: "plant/a",
			Fields: []v3.ReadField{{
				Key:      "f1",
				Name:     "level",
				Channel:  7,
				Pointer:  "/level",
				DataType: telem.Float64T,
			}},
		}}}}
		ic, ok := MustSucceed2(v3.ResolveReadConfig(
			ctx, nil, channels, "Plant", &cfg, func(_, _ string) {},
		))
		Expect(ok).To(BeTrue())
		Expect(cfg.Inline).To(BeNil())
		Expect(cfg.Entries).To(HaveLen(1))
		plain, isPlain := cfg.Entries[0].Variant.(v3.PlainReadEntry)
		Expect(isPlain).To(BeTrue())
		Expect(plain.ICD).To(Equal(ic.Key))
		Expect(plain.Messages).To(HaveLen(1))
		Expect(plain.Messages[0].Message).To(Equal(ic.Messages[0].Key))
		Expect(plain.Messages[0].Bindings[0].Channel).To(Equal(channel.Key(7)))
	})
})
