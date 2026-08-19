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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/security"
	secmock "github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	panelversions "github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	projectv0 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Layer", func() {
	// Regression: the composition pass converts data staged by services that open
	// after the panel table (project layouts, task re-key mapping). Running the
	// conversion inside the panel table's own chain silently produced zero panels,
	// because the chain ran before the producers staged anything.
	It(
		"Should adopt a legacy workspace layout as a panel after all tables open",
		func(ctx SpecContext) {
			node := mock.NewNode(ctx)
			db := node.DB
			lpKey := uuid.NewString()
			lpID := ontology.ID{Type: ontology.ResourceTypeLineplot, Key: lpKey}
			Expect(gorp.WrapWriter[string, ontology.Resource](db).
				Set(ctx, ontology.Resource{ID: lpID})).To(Succeed())
			ws := projectv0.Workspace{
				Key:  uuid.New(),
				Name: "TPC",
				Layout: msgpack.EncodedJSON{
					"mosaics": map[string]any{
						"main": map[string]any{
							"root": map[string]any{
								"key": 1,
								"tabs": []any{
									map[string]any{"tabKey": lpKey},
								},
							},
						},
					},
					"layouts": map[string]any{
						lpKey: map[string]any{
							"key":      lpKey,
							"type":     "lineplot",
							"name":     "Pressures",
							"location": "mosaic",
						},
						"main": map[string]any{
							"key":      "main",
							"type":     "main",
							"name":     "Main",
							"location": "window",
						},
					},
				},
			}
			Expect(gorp.WrapWriter[projectv0.Key, projectv0.Workspace](db).
				Set(ctx, ws)).To(Succeed())

			sec := MustSucceed(security.NewProvider(security.ProviderConfig{
				Insecure: new(true),
				KeySize:  secmock.SmallKeySize,
			}))
			MustOpen(service.OpenLayer(ctx, service.LayerConfig{
				Distribution: node.Layer,
				Security:     sec,
				Storage:      node.Storage,
			}))

			seq, closer := MustSucceed2(gorp.
				WrapReader[panelversions.Key, panelversions.Panel](db).
				OpenNexter(ctx))
			DeferClose(closer)
			var panels []panelversions.Panel
			for p := range seq {
				panels = append(panels, p)
			}
			Expect(panels).To(HaveLen(1))
			Expect(panels[0].Name).To(Equal("Main"))
			lf, ok := panels[0].Root.Variant.(panelversions.LeafNode)
			Expect(ok).To(BeTrue())
			Expect(lf.Tabs).To(HaveLen(1))
			rt, ok := lf.Tabs[0].Variant.(panelversions.ResourceTab)
			Expect(ok).To(BeTrue())
			Expect(rt.Resource).To(Equal(lpID))

			By("Parenting the panel under the migrated project")
			projectID := ontology.ID{
				Type: ontology.ResourceTypeProject,
				Key:  ws.Key.String(),
			}
			rel := ontology.Relationship{
				From: projectID,
				Type: ontology.RelationshipTypeParentOf,
				To:   panels[0].OntologyID(),
			}
			Expect(MustSucceed(gorp.NewRetrieve[string, ontology.Relationship]().
				Where(gorp.MatchKeys[string, ontology.Relationship](rel.GorpKey())).
				Exists(ctx, db))).To(BeTrue())
		},
	)
})
