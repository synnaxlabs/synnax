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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	groupv0 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v0"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	rangerv0 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
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
				Exec(ctx, nil)).To(Succeed())
			Expect(g.Scope).To(Equal(resource.TypeChannel))
			Expect(g.Parent).To(BeNil())
			var r ranger.Range
			Expect(layer.Ranger.NewRetrieve().
				Where(ranger.MatchKeys(child.Key)).
				Entry(&r).
				Exec(ctx, nil)).To(Succeed())
			Expect(r.Parent).To(HaveValue(Equal(parent.Key)))

			By("Deleting the built-in root group")
			Expect(layer.Group.NewRetrieve().
				Where(group.MatchKeys(channels.Key)).
				Exists(ctx, nil)).To(BeFalse())

			By("Deleting every resource and relationship")
			Expect(resources(ctx, node.DB)).To(BeEmpty())
			Expect(relationships(ctx, node.DB)).To(BeEmpty())
		},
	)

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
			Exists(ctx, nil)).To(BeTrue())
		Expect(resources(ctx, node.DB)).To(HaveLen(1))
		Expect(relationships(ctx, node.DB)).To(ConsistOf(rel))
	})

	Describe("Channels", func() {
		names := []string{
			"sy_ontology_resource_set",
			"sy_ontology_resource_delete",
			"sy_ontology_relationship_set",
			"sy_ontology_relationship_delete",
		}
		retrieve := func(ctx SpecContext, l *service.Layer) []channel.Channel {
			GinkgoHelper()
			var chs []channel.Channel
			Expect(l.Channel.NewRetrieve().
				Where(channel.MatchNames(names...)).
				Entries(&chs).
				Exec(ctx, nil)).To(Succeed())
			return chs
		}

		It(
			"Should delete the ontology channels of an upgraded store",
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
				Expect(first.Channel.NewWriter(nil).CreateMany(ctx, &chs)).To(Succeed())
				Expect(retrieve(ctx, first)).To(HaveLen(len(names)))
				Expect(first.Close()).To(Succeed())
				second := MustSucceed(svctestutil.OpenLayer(ctx, n))
				Expect(retrieve(ctx, second)).To(BeEmpty())
				Expect(second.Close()).To(Succeed())
				By("Opening again with no channels to delete")
				Expect(retrieve(ctx, MustOpen(svctestutil.OpenLayer(ctx, n)))).
					To(BeEmpty())
			},
		)
	})
})
