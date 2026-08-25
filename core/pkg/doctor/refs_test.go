// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/doctor"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ni"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	rangekv "github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Referential checks", func() {
	Describe("Ontology", func() {
		It("Should report a resource of an unknown type", func(ctx SpecContext) {
			c := createCore(ctx)
			id := ontology.ID{Type: "banana", Key: "1"}
			write[string, ontology.Resource](ctx, c, ontology.Resource{ID: id})
			r := run(ctx, c)
			Expect(messages(r, doctor.CheckResourceType)).To(ConsistOf(
				"resource has an unknown type",
			))
			Expect(findings(r, doctor.CheckResourceType)[0].Subject).
				To(Equal(id.String()))
		})

		It("Should report a resource whose entity is gone", func(ctx SpecContext) {
			c := createCore(ctx)
			id := ontology.ID{Type: ontology.ResourceTypeLabel, Key: uuid.NewString()}
			write[string, ontology.Resource](ctx, c, ontology.Resource{ID: id})
			r := run(ctx, c)
			Expect(messages(r, doctor.CheckResourceOrphan)).To(ConsistOf(
				"resource has no backing entity",
			))
		})

		It("Should report a relationship pointing at a deleted resource", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			rel := ontology.Relationship{
				From: ontology.ID{Type: ontology.ResourceTypeLabel, Key: "gone"},
				To:   ontology.ID{Type: ontology.ResourceTypeLabel, Key: "also-gone"},
				Type: ontology.RelationshipTypeParentOf,
			}
			write[string, ontology.Relationship](ctx, c, rel)
			r := run(ctx, c)
			found := findings(r, doctor.CheckRelationshipEndpoint)
			Expect(found).To(HaveLen(1))
			Expect(found[0].Message).To(ContainSubstring("2 occurrences"))
			Expect(found[0].Subject).To(Equal(rel.GorpKey()))
		})

		It("Should report a relationship key that cannot be parsed", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			rel := ontology.Relationship{
				From: ontology.ID{Type: ontology.ResourceTypeLabel, Key: "a"},
				To:   ontology.ID{Type: ontology.ResourceTypeLabel, Key: "b"},
				Type: ontology.RelationshipTypeParentOf,
			}
			write[string, ontology.Relationship](ctx, c, rel)
			key := append([]byte("gorp.Relationship"), []byte("mangled")...)
			value := readEntry(ctx, c, "gorp.Relationship"+rel.GorpKey())
			Expect(c.Storage.KV.Set(ctx, key, value)).To(Succeed())
			r := run(ctx, c)
			Expect(messages(r, doctor.CheckRelationshipKey)).To(ConsistOf(
				"relationship key cannot be parsed",
			))
		})
	})

	Describe("Ranges", func() {
		It("Should report an alias with no range and no channel", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			write[string, alias.Alias](ctx, c, alias.Alias{
				Range:   ranger.Key(uuid.New()),
				Channel: channel.Key(9999),
				Alias:   "ghost",
			})
			r := run(ctx, c)
			Expect(messages(r, doctor.CheckAlias)).To(ConsistOf(
				"alias references a deleted range",
				"alias references a deleted channel",
			))
		})

		It("Should report a key-value pair with no range", func(ctx SpecContext) {
			c := createCore(ctx)
			write[string, rangekv.Pair](ctx, c, rangekv.Pair{
				Range: ranger.Key(uuid.New()),
				Key:   "operator",
				Value: "sam",
			})
			r := run(ctx, c)
			Expect(messages(r, doctor.CheckRangeKV)).To(ConsistOf(
				"key-value pair references a deleted range",
			))
		})
	})

	Describe("Tasks", func() {
		It("Should report a task with a deleted rack and no configuration", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			write[task.Key, task.Task](ctx, c, task.Task{
				Key:  uuid.New(),
				Rack: rack.NewKey(1, 42),
				Name: "ghost task",
				Type: "ni_scanner",
			})
			r := run(ctx, c)
			Expect(messages(r, doctor.CheckRack)).To(ConsistOf(
				"task references a deleted rack",
			))
			Expect(messages(r, doctor.CheckTaskConfig)).To(ConsistOf(
				"task has no stored configuration",
			))
		})

		It("Should report a configuration record with no task", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			cfg := ni.ScanConfig{}
			cfg.Key = uuid.New()
			write[uuid.UUID, ni.ScanConfig](ctx, c, cfg)
			r := run(ctx, c)
			found := findings(r, doctor.CheckTaskConfig)
			Expect(found).To(HaveLen(1))
			Expect(found[0].Message).To(Equal("config record has no task"))
			Expect(found[0].Subject).To(Equal(cfg.Key.String()))
		})
	})

	Describe("Users", func() {
		It("Should report credentials with no user", func(ctx SpecContext) {
			c := createCore(ctx)
			write[string, auth.SecureCredentials](ctx, c, auth.SecureCredentials{
				Username: "ghost",
				Password: []byte("hashed"),
			})
			r := run(ctx, c)
			found := findings(r, doctor.CheckCredentials)
			Expect(found).To(HaveLen(1))
			Expect(found[0].Subject).To(Equal("ghost"))
		})
	})

	Describe("Channels", func() {
		It("Should report a channel whose index channel is gone", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			index := createChannel(ctx, c, channel.Channel{
				Name:     "time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			data := createChannel(ctx, c, channel.Channel{
				Name:       "pressure",
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			})
			Expect(data.LocalIndex).ToNot(BeZero())
			deleteEntry[channel.Key, channel.Channel](ctx, c, index.Key())
			r := run(ctx, c)
			Expect(messages(r, doctor.CheckChannelIndex)).To(ConsistOf(
				"channel references a deleted index channel",
			))
		})

		It("Should report metadata that disagrees with Cesium", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			ch := createChannel(ctx, c, channel.Channel{
				Name:     "time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			ch.DataType = telem.Int64T
			write[channel.Key, channel.Channel](ctx, c, ch)
			r := run(ctx, c)
			Expect(messages(r, doctor.CheckChannelMeta)).To(ConsistOf(
				"data type disagrees with Cesium",
			))
		})
	})

	Describe("Access", func() {
		It("Should report a policy object naming a deleted resource", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			write[policy.Key, policy.Policy](ctx, c, policy.Policy{
				Key:  uuid.New(),
				Name: "ghost policy",
				Objects: []ontology.ID{
					{Type: ontology.ResourceTypeChannel, Key: "9999"},
					{Type: ontology.ResourceTypeChannel},
				},
			})
			r := run(ctx, c)
			found := findings(r, doctor.CheckPolicyObject)
			Expect(found).To(HaveLen(1))
			Expect(found[0].Severity).To(Equal(doctor.SeverityInfo))
			Expect(found[0].Message).ToNot(ContainSubstring("occurrences"))
		})
	})

	Describe("Panels", func() {
		It("Should report a tab displaying a deleted resource", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			missing := ontology.ID{Type: ontology.ResourceTypeLog, Key: "gone"}
			write[panel.Key, panel.Panel](ctx, c, panel.Panel{
				Key:  uuid.New(),
				Name: "ghost panel",
				Root: panel.Node{Variant: panel.SplitNode{
					First: panel.Node{Variant: panel.LeafNode{Tabs: []panel.Tab{
						{Variant: panel.ResourceTab{Resource: missing}},
					}}},
					Last: panel.Node{Variant: panel.LeafNode{}},
				}},
			})
			r := run(ctx, c)
			found := findings(r, doctor.CheckPanelTab)
			Expect(found).To(HaveLen(1))
			Expect(found[0].Subject).To(ContainSubstring(missing.String()))
		})
	})
})
