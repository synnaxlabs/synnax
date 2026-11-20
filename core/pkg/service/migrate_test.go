// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"context"
	"fmt"
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv0 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	groupv0 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v0"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	rangerv0 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func parentOf(from, to resource.ID) ontology.Relationship {
	return ontology.Relationship{
		From: from,
		Type: ontology.RelationshipTypeParentOf,
		To:   to,
	}
}

func resources(ctx context.Context, db *gorp.DB) []ontology.Resource {
	GinkgoHelper()
	var out []ontology.Resource
	Expect(gorp.NewRetrieve[string, ontology.Resource]().
		Entries(&out).
		Exec(ctx, db)).To(Succeed())
	return out
}

func relationships(ctx context.Context, db *gorp.DB) []ontology.Relationship {
	GinkgoHelper()
	var out []ontology.Relationship
	Expect(gorp.NewRetrieve[string, ontology.Relationship]().
		Entries(&out).
		Exec(ctx, db)).To(Succeed())
	return out
}

var _ = Describe("Ontology drop", func() {
	It(
		"Should convert the edges of an upgraded store, then drop the ontology",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			legacyDB := gorp.Wrap(node.DB.KV(), gorp.WithCodec(msgpack.Codec))
			rootID := resource.ID{Type: resource.TypeBuiltin, Key: "root"}
			channels := groupv0.Group{Key: uuid.New(), Name: "Channels"}
			sensors := groupv0.Group{Key: uuid.New(), Name: "Sensors"}
			Expect(gorp.WrapWriter[groupv0.Key, groupv0.Group](legacyDB).
				Set(ctx, channels, sensors)).To(Succeed())
			tr := telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			}
			parent := rangerv0.Range{Key: uuid.New(), Name: "Parent", TimeRange: tr}
			child := rangerv0.Range{Key: uuid.New(), Name: "Child", TimeRange: tr}
			Expect(gorp.WrapWriter[rangerv0.Key, rangerv0.Range](legacyDB).
				Set(ctx, parent, child)).To(Succeed())
			channelsID := resource.ID{
				Type: resource.TypeGroup,
				Key:  channels.Key.String(),
			}
			sensorsID := resource.ID{
				Type: resource.TypeGroup,
				Key:  sensors.Key.String(),
			}
			Expect(gorp.WrapWriter[string, ontology.Resource](legacyDB).Set(
				ctx,
				ontology.Resource{ID: rootID},
				ontology.Resource{ID: channelsID},
				ontology.Resource{ID: sensorsID},
				ontology.Resource{ID: parent.ResourceID()},
				ontology.Resource{ID: child.ResourceID()},
			)).To(Succeed())
			Expect(gorp.WrapWriter[string, ontology.Relationship](legacyDB).Set(
				ctx,
				parentOf(rootID, channelsID),
				parentOf(channelsID, sensorsID),
				parentOf(parent.ResourceID(), child.ResourceID()),
			)).To(Succeed())

			layer := MustOpen(svctestutil.OpenLayer(ctx, node))

			By("Converting the group and range edges into fields")
			var g group.Group
			Expect(layer.Group.NewRetrieve().
				Where(group.MatchKeys(sensors.Key)).
				Entry(&g).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(g.Scope).To(Equal(resource.TypeChannel))
			Expect(g.Parent).To(BeNil())
			var r ranger.Range
			Expect(layer.Ranger.NewRetrieve().
				Where(ranger.MatchKeys(child.Key)).
				Entry(&r).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(r.Parent).To(HaveValue(Equal(parent.Key)))

			By("Deleting the built-in root group")
			Expect(layer.Group.NewRetrieve().
				Where(group.MatchKeys(channels.Key)).
				Exists(ctx, node.DB)).To(BeFalse())

			By("Deleting every resource and relationship")
			Expect(resources(ctx, node.DB)).To(BeEmpty())
			Expect(relationships(ctx, node.DB)).To(BeEmpty())
		},
	)

	It("Should log each label edge it drops", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		legacyDB := gorp.Wrap(node.DB.KV(), gorp.WithCodec(msgpack.Codec))
		labelID := resource.ID{Type: resource.TypeLabel, Key: uuid.New().String()}
		taskID := resource.ID{Type: resource.TypeTask, Key: uuid.New().String()}
		Expect(gorp.WrapWriter[string, ontology.Relationship](legacyDB).Set(
			ctx,
			ontology.Relationship{
				From: taskID,
				Type: ontology.RelationshipTypeLabeledBy,
				To:   labelID,
			},
		)).To(Succeed())
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		MustOpen(svctestutil.OpenLayer(ctx, node, service.LayerConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
		}))
		dropped := logs.FilterMessage(
			"dropped a label from a resource that cannot hold labels",
		).All()
		Expect(dropped).To(HaveLen(1))
		Expect(dropped[0].ContextMap()).To(Equal(map[string]any{
			"label":    labelID.String(),
			"resource": taskID.String(),
		}))
		Expect(relationships(ctx, node.DB)).To(BeEmpty())
	})

	It("Should not run again on a later open", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		Expect(MustSucceed(svctestutil.OpenLayer(ctx, node)).Close()).To(Succeed())
		builtin := group.Group{Name: "Builtin", Scope: resource.TypeBuiltin}
		builtin.Key = uuid.New()
		rel := parentOf(
			resource.ID{Type: resource.TypeBuiltin, Key: "root"},
			builtin.ResourceID(),
		)
		Expect(gorp.WrapWriter[group.Key, group.Group](node.DB).Set(ctx, builtin)).
			To(Succeed())
		Expect(gorp.WrapWriter[string, ontology.Resource](node.DB).
			Set(ctx, ontology.Resource{ID: builtin.ResourceID()})).To(Succeed())
		Expect(gorp.WrapWriter[string, ontology.Relationship](node.DB).Set(ctx, rel)).
			To(Succeed())
		layer := MustOpen(svctestutil.OpenLayer(ctx, node))
		Expect(layer.Group.NewRetrieve().
			Where(group.MatchKeys(builtin.Key)).
			Exists(ctx, node.DB)).To(BeTrue())
		Expect(resources(ctx, node.DB)).To(HaveLen(1))
		Expect(relationships(ctx, node.DB)).To(ConsistOf(rel))
	})
})

var _ = Describe("Retired channels", func() {
	names := []string{
		"sy_ontology_resource_set",
		"sy_ontology_resource_delete",
		"sy_ontology_relationship_set",
		"sy_ontology_relationship_delete",
		"sy_range_alias_set",
		"sy_range_alias_delete",
		"sy_range_alias_action_set",
	}
	retrieve := func(
		ctx SpecContext,
		db *gorp.DB,
		l *service.Layer,
	) []channel.Channel {
		GinkgoHelper()
		var chs []channel.Channel
		Expect(l.Channel.NewRetrieve().
			Where(channel.MatchNames(names...)).
			Entries(&chs).
			Exec(ctx, db)).To(Succeed())
		return chs
	}

	It(
		"Should delete the retired channels of an upgraded store",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			first := MustSucceed(svctestutil.OpenLayer(ctx, n))
			chs := make([]channel.Channel, len(names))
			for i, name := range names {
				chs[i] = channel.Channel{
					Name:        name,
					DataType:    telem.StringT,
					Virtual:     true,
					Internal:    true,
					Leaseholder: node.KeyFree,
				}
			}
			Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return first.Channel.NewWriter(tx).CreateMany(ctx, &chs)
			})).To(Succeed())
			Expect(retrieve(ctx, n.DB, first)).To(HaveLen(len(names)))
			Expect(first.Close()).To(Succeed())
			second := MustSucceed(svctestutil.OpenLayer(ctx, n))
			Expect(retrieve(ctx, n.DB, second)).To(BeEmpty())
			Expect(second.Close()).To(Succeed())
			By("Opening again with no channels to delete")
			Expect(retrieve(ctx, n.DB, MustOpen(svctestutil.OpenLayer(ctx, n)))).
				To(BeEmpty())
		},
	)
})

// storedAlias is the shape that releases before v0.62 stored a range alias in.
type storedAlias struct {
	Range   uuid.UUID
	Channel uint32
	Alias   string
}

func (a storedAlias) GorpKey() string {
	return a.Range.String() + "---" + strconv.Itoa(int(a.Channel))
}

func (storedAlias) CustomTypeName() string { return "Alias" }

var _ = Describe("Range alias drop", func() {
	It("Should delete the range aliases of an upgraded store", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		legacyDB := gorp.Wrap(n.DB.KV(), gorp.WithCodec(msgpack.Codec))
		Expect(gorp.WrapWriter[string, storedAlias](legacyDB).Set(
			ctx,
			storedAlias{Range: uuid.New(), Channel: 1, Alias: "pressure"},
			storedAlias{Range: uuid.New(), Channel: 2, Alias: "temperature"},
		)).To(Succeed())
		MustOpen(svctestutil.OpenLayer(ctx, n))
		var aliases []storedAlias
		Expect(gorp.NewRetrieve[string, storedAlias]().
			Entries(&aliases).
			Exec(ctx, legacyDB)).To(Succeed())
		Expect(aliases).To(BeEmpty())
	})

	It("Should log each range alias it deletes", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		legacyDB := gorp.Wrap(n.DB.KV(), gorp.WithCodec(msgpack.Codec))
		alias := storedAlias{Range: uuid.New(), Channel: 1, Alias: "pressure"}
		Expect(gorp.WrapWriter[string, storedAlias](legacyDB).Set(ctx, alias)).
			To(Succeed())
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		MustOpen(svctestutil.OpenLayer(ctx, n, service.LayerConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
		}))
		deleted := logs.FilterMessage("deleted an entry").All()
		Expect(deleted).To(HaveLen(1))
		Expect(deleted[0].ContextMap()).To(Equal(map[string]any{
			"type": "Alias",
			"key":  alias.Range.String() + "---1",
		}))
	})
})

var _ = Describe("Metrics upgrade", func() {
	It("Should keep every metrics channel in the Metrics group", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		legacyDB := gorp.Wrap(n.DB.KV(), gorp.WithCodec(msgpack.Codec))
		prefix := fmt.Sprintf("sy_node_%s_metrics_", n.Identity.HostKey)
		idx := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{{
			Name:     prefix + "time",
			DataType: telem.TimestampT,
			IsIndex:  true,
		}}))[0]
		dataNames := []string{
			"cpu_percentage", "mem_percentage", "ts_size_gb", "kv_size_gb",
		}
		data := make([]distchannel.Channel, len(dataNames))
		for i, name := range dataNames {
			data[i] = distchannel.Channel{
				Name:       prefix + name,
				DataType:   telem.Float32T,
				LocalIndex: idx.LocalKey,
			}
		}
		stored := append(
			[]distchannel.Channel{idx},
			MustSucceed(n.Channel.Allocate(ctx, data))...,
		)
		Expect(n.Channel.Create(ctx, stored)).To(Succeed())
		calcIdx := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{{
			Name:        prefix + "total_size_gb_time",
			Leaseholder: node.KeyFree,
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Virtual:     true,
		}}))[0]
		calc := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{{
			Name:        prefix + "total_size_gb",
			Leaseholder: node.KeyFree,
			DataType:    telem.Float32T,
			LocalIndex:  calcIdx.LocalKey,
			Virtual:     true,
		}}))[0]
		var (
			root      = groupv0.Group{Key: uuid.New(), Name: "Channels"}
			metrics   = groupv0.Group{Key: uuid.New(), Name: "Metrics"}
			rootID    = resource.ID{Type: resource.TypeGroup, Key: root.Key.String()}
			metricsID = resource.ID{Type: resource.TypeGroup, Key: metrics.Key.String()}
			resources = append(
				make([]ontology.Resource, 0, 4+len(stored)),
				ontology.Resource{ID: rootID},
				ontology.Resource{ID: metricsID},
			)
			edges = []ontology.Relationship{
				parentOf(ontology.RootID, rootID),
				parentOf(rootID, metricsID),
			}
			records = make([]channelv0.Channel, 0, len(stored)+2)
		)
		for _, ch := range append(stored, calcIdx, calc) {
			rec := channelv0.Channel{
				Name:        ch.Name,
				Leaseholder: ch.Leaseholder,
				DataType:    ch.DataType,
				IsIndex:     ch.IsIndex,
				LocalKey:    ch.LocalKey,
				LocalIndex:  ch.LocalIndex,
				Virtual:     ch.Virtual,
			}
			if ch.Name == calc.Name {
				rec.Expression = fmt.Sprintf(
					"return %sts_size_gb + %skv_size_gb", prefix, prefix,
				)
			}
			records = append(records, rec)
			id := resource.ID{Type: resource.TypeChannel, Key: ch.Key().String()}
			resources = append(resources, ontology.Resource{ID: id})
			// 0.60 gave the index of a calculated metric no group edge.
			if ch.Name != calcIdx.Name {
				edges = append(edges, parentOf(metricsID, id))
			}
		}
		Expect(gorp.WrapWriter[channelv0.Key, channelv0.Channel](legacyDB).
			Set(ctx, records...)).To(Succeed())
		Expect(gorp.WrapWriter[groupv0.Key, groupv0.Group](legacyDB).
			Set(ctx, root, metrics)).To(Succeed())
		Expect(gorp.WrapWriter[string, ontology.Resource](n.DB).
			Set(ctx, resources...)).To(Succeed())
		Expect(gorp.WrapWriter[string, ontology.Relationship](n.DB).
			Set(ctx, edges...)).To(Succeed())
		layer := MustOpen(svctestutil.OpenLayer(ctx, n))
		var chs []channel.Channel
		Expect(layer.Channel.NewRetrieve().
			Where(channel.MatchNames(lo.Map(
				records,
				func(r channelv0.Channel, _ int) string { return r.Name },
			)...)).
			Entries(&chs).
			Exec(ctx, n.DB)).To(Succeed())
		Expect(chs).To(HaveLen(len(records)))
		for _, ch := range chs {
			Expect(ch.Group).To(HaveValue(Equal(metrics.Key)), ch.Name)
		}
	})
})

var _ = Describe("Group drop report", func() {
	It("Should report each channel that the upgrade removed from a group",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			legacyDB := gorp.Wrap(n.DB.KV(), gorp.WithCodec(msgpack.Codec))
			chs := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{
				{
					Name:        "oil_temp",
					Leaseholder: node.KeyFree,
					DataType:    telem.Float32T,
					Virtual:     true,
				},
				{
					Name:        "water_temp",
					Leaseholder: node.KeyFree,
					DataType:    telem.Float32T,
					Virtual:     true,
				},
				{
					Name:        "flow_rate",
					Leaseholder: node.KeyFree,
					DataType:    telem.Float32T,
					Virtual:     true,
				},
			}))
			var (
				channels = groupv0.Group{Key: uuid.New(), Name: "Channels"}
				devices  = groupv0.Group{Key: uuid.New(), Name: "Devices"}
				// Cryomech is first by key and by bytes, so only a pick by name that
				// ignores case keeps alpha.
				cryomech = groupv0.Group{
					Key:  uuid.MustParse("00000000-0000-4000-8000-000000000001"),
					Name: "Cryomech",
				}
				alpha = groupv0.Group{
					Key:  uuid.MustParse("ffffffff-ffff-4fff-bfff-ffffffffffff"),
					Name: "alpha",
				}
				compressors = groupv0.Group{Key: uuid.New(), Name: "Compressors"}
				// Mixed has no parent and holds two kinds of members, so the upgrade
				// deletes it.
				mixed = groupv0.Group{Key: uuid.New(), Name: "Mixed"}
				id    = func(g groupv0.Group) resource.ID {
					return resource.ID{Type: resource.TypeGroup, Key: g.Key.String()}
				}
				oil = resource.ID{
					Type: resource.TypeChannel,
					Key:  chs[0].Key().String(),
				}
				water = resource.ID{
					Type: resource.TypeChannel,
					Key:  chs[1].Key().String(),
				}
				flow = resource.ID{
					Type: resource.TypeChannel,
					Key:  chs[2].Key().String(),
				}
				groups = []groupv0.Group{
					channels,
					devices,
					cryomech,
					alpha,
					compressors,
					mixed,
				}
				res = append(
					make([]ontology.Resource, 0, 3+len(groups)),
					ontology.Resource{ID: oil},
					ontology.Resource{ID: water},
					ontology.Resource{ID: flow},
				)
				recs = make([]channelv0.Channel, 0, len(chs))
			)
			for _, ch := range chs {
				recs = append(recs, channelv0.Channel{
					Name:        ch.Name,
					Leaseholder: ch.Leaseholder,
					DataType:    ch.DataType,
					LocalKey:    ch.LocalKey,
					Virtual:     ch.Virtual,
				})
			}
			for _, g := range groups {
				res = append(res, ontology.Resource{ID: id(g)})
			}
			Expect(gorp.WrapWriter[channelv0.Key, channelv0.Channel](legacyDB).
				Set(ctx, recs...)).To(Succeed())
			Expect(gorp.WrapWriter[groupv0.Key, groupv0.Group](legacyDB).
				Set(ctx, groups...)).To(Succeed())
			Expect(gorp.WrapWriter[string, ontology.Resource](n.DB).
				Set(ctx, res...)).To(Succeed())
			Expect(gorp.WrapWriter[string, ontology.Relationship](n.DB).Set(
				ctx,
				parentOf(ontology.RootID, id(channels)),
				parentOf(ontology.RootID, id(devices)),
				parentOf(id(channels), id(cryomech)),
				parentOf(id(channels), id(alpha)),
				parentOf(id(devices), id(compressors)),
				parentOf(id(cryomech), oil),
				parentOf(id(alpha), oil),
				parentOf(id(compressors), water),
				parentOf(id(compressors), resource.ID{
					Type: resource.TypeChannel,
					Key:  "999999",
				}),
				parentOf(id(mixed), flow),
				parentOf(id(mixed), resource.ID{
					Type: resource.TypeSchematic,
					Key:  uuid.New().String(),
				}),
			)).To(Succeed())
			layer := MustOpen(svctestutil.OpenLayer(ctx, n))

			By("Keeping a channel of several groups in the first group by name")
			var migrated []channel.Channel
			Expect(layer.Channel.NewRetrieve().
				Where(channel.MatchNames("oil_temp", "water_temp")).
				Entries(&migrated).
				Exec(ctx, n.DB)).To(Succeed())
			groupOf := make(map[string]*group.Key, len(migrated))
			for _, ch := range migrated {
				groupOf[ch.Name] = ch.Group
			}
			Expect(groupOf["oil_temp"]).To(HaveValue(Equal(alpha.Key)))
			Expect(groupOf["water_temp"]).To(BeNil())

			By("Writing a warning status for each removal")
			var statuses []status.Status[any]
			Expect(layer.Status.NewRetrieve[any]().
				Entries(&statuses).
				Exec(ctx, n.DB)).To(Succeed())
			type report struct{ name, message, description string }
			var reports []report
			for _, s := range statuses {
				if s.Message == "Removed channel from group" {
					Expect(s.Variant).To(Equal(status.VariantWarning))
					reports = append(reports, report{s.Name, s.Message, s.Description})
				}
			}
			Expect(reports).To(ConsistOf(
				report{
					name:    "Cryomech",
					message: "Removed channel from group",
					description: `A channel can now be in only one group, so the ` +
						`upgrade removed channel "oil_temp" from group "Cryomech". It ` +
						`kept the channel in the group that comes first by name.`,
				},
				report{
					name:    "Compressors",
					message: "Removed channel from group",
					description: `The upgrade removed channel "water_temp" from group ` +
						`"Compressors", because that group cannot hold a channel.`,
				},
				report{
					name:    "Compressors",
					message: "Removed channel from group",
					description: `The upgrade removed channel "999999" from group ` +
						`"Compressors", because that group cannot hold a channel.`,
				},
				report{
					name:    "Mixed",
					message: "Removed channel from group",
					description: `The upgrade deleted group "Mixed", so channel ` +
						`"flow_rate" is no longer in it.`,
				},
			))
		},
	)
})
