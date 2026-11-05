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
	"encoding/json/v2"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	authv0 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	panelversions "github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	projectv0 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v0"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	userv0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"golang.org/x/crypto/bcrypt"
)

var _ = Describe("Layer", func() {
	It("Should cap external channels at the license's limit", func(ctx SpecContext) {
		signer := svctestutil.NewSigner()
		lic := svctestutil.NewLicense()
		lic.Channels = 50
		layer := MustOpen(
			svctestutil.OpenLayer(ctx, testutil.NewNode(ctx), service.LayerConfig{
				License: license.ServiceConfig{
					Key:     signer.Sign(lic),
					Anchors: signer.Anchors,
				},
			}),
		)
		var existing []channel.Channel
		Expect(layer.Channel.NewRetrieve().Entries(&existing).Exec(ctx, nil)).
			To(Succeed())
		external := lo.CountBy(existing, func(ch channel.Channel) bool {
			return !ch.Internal && !ch.Virtual
		})
		newIndices := func(n int) []channel.Channel {
			return lo.Times(n, func(int) channel.Channel {
				return channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.TimestampT,
					IsIndex:     true,
					Leaseholder: 1,
				}
			})
		}
		w := layer.Channel.NewWriter(nil)
		Expect(w.CreateMany(ctx, new(newIndices(50-external)))).To(Succeed())
		Expect(w.CreateMany(ctx, new(newIndices(1)))).To(And(
			MatchError(license.ErrTooMany),
			MatchError(ContainSubstring("limit is 50 channels")),
		))
	})

	DescribeTable(
		"Should publish NI resource changes on signal channels",
		func(
			ctx SpecContext,
			name string,
			create func(context.Context, *service.Layer) uuid.UUID,
			remove func(context.Context, *service.Layer, uuid.UUID) error,
		) {
			layer := MustOpen(svctestutil.OpenLayer(ctx, testutil.NewNode(ctx)))
			openStreamer := func(
				name string,
			) confluence.Outlet[framer.StreamerResponse] {
				GinkgoHelper()
				var ch channel.Channel
				Expect(layer.Channel.NewRetrieve().
					Where(channel.MatchNames(name)).
					Entry(&ch).
					Exec(ctx, nil)).To(Succeed())
				Expect(ch.Virtual).To(BeTrue())
				Expect(ch.Internal).To(BeTrue())
				streamer := MustSucceed(layer.Framer.NewStreamer(
					ctx,
					framer.StreamerConfig{
						Keys:        channel.Keys{ch.Key()},
						SendOpenAck: true,
					},
				))
				requests, responses := confluence.Attach(streamer, 2)
				sCtx, cancel := signal.Isolated()
				shutdown := signal.NewHardShutdown(sCtx, cancel)
				streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				DeferCleanup(func() {
					requests.Close()
					confluence.Drain(responses)
					Expect(shutdown.Close()).To(Succeed())
				})
				Eventually(responses.Outlet(), 5*time.Second).Should(Receive())
				return responses
			}
			sets := openStreamer("sy_ni_" + name + "_set")
			deletes := openStreamer("sy_ni_" + name + "_delete")

			key := create(ctx, layer)
			var res framer.StreamerResponse
			Eventually(sets.Outlet(), 5*time.Second).Should(Receive(&res))
			var written struct {
				Key uuid.UUID `json:"key"`
			}
			for sample := range res.Frame.SeriesAt(0).Samples() {
				Expect(json.Unmarshal(sample, &written)).To(Succeed())
			}
			Expect(written.Key).To(Equal(key))

			Expect(remove(ctx, layer, key)).To(Succeed())
			Eventually(deletes.Outlet(), 5*time.Second).Should(Receive(&res))
			var removed []uuid.UUID
			for sample := range res.Frame.SeriesAt(0).Samples() {
				removed = append(removed, uuid.UUID(sample))
			}
			Expect(removed).To(ContainElement(key))
		},
		Entry(
			"chassis",
			"chassis",
			func(ctx context.Context, layer *service.Layer) uuid.UUID {
				GinkgoHelper()
				c := chassis.Chassis{
					Rack:         layer.Rack.EmbeddedKey,
					Name:         "PXIe-1071",
					ResourceName: "Mod1",
					Model:        "PXIe-1071",
				}
				Expect(layer.NI.Chassis.NewWriter(nil).Create(ctx, &c)).To(Succeed())
				return c.Key
			},
			func(ctx context.Context, layer *service.Layer, key uuid.UUID) error {
				return layer.NI.Chassis.NewWriter(nil).Delete(ctx, key)
			},
		),
		Entry(
			"device",
			"device",
			func(ctx context.Context, layer *service.Layer) uuid.UUID {
				GinkgoHelper()
				d := device.Device{Rack: layer.Rack.EmbeddedKey, Name: "PXI-6255"}
				Expect(layer.NI.Devices.NewWriter(nil).Create(ctx, &d)).To(Succeed())
				return d.Key
			},
			func(ctx context.Context, layer *service.Layer, key uuid.UUID) error {
				return layer.NI.Devices.NewWriter(nil).Delete(ctx, key)
			},
		),
	)

	It(
		"Should end root sessions only when the root password changes",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			open := func(pwd string) *service.Layer {
				GinkgoHelper()
				return MustSucceed(svctestutil.OpenLayer(ctx, node, service.LayerConfig{
					RootCredentials: auth.Credentials{Username: "root", Password: pwd},
				}))
			}
			first := open("p1")
			_, token := MustSucceed2(first.Auth.Login(ctx, auth.Proof{
				Variant: auth.PasswordProof{
					Credentials: auth.Credentials{Username: "root", Password: "p1"},
				},
			}))
			Expect(first.Close()).To(Succeed())
			second := open("p1")
			MustSucceed(second.Auth.Authenticate(ctx, token))
			Expect(second.Close()).To(Succeed())
			third := open("p2")
			Expect(third.Auth.Authenticate(ctx, token)).Error().
				To(MatchError(auth.ErrInvalidToken))
			Expect(third.Close()).To(Succeed())
		},
	)

	It(
		"Should log in a user whose password a Core before 0.59 stored",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			alice := userv0.User{Key: uuid.New(), Username: "alice"}
			hash := MustSucceed(
				bcrypt.GenerateFromPassword([]byte("p"), bcrypt.MinCost),
			)
			Expect(gorp.WrapWriter[string, ontology.Resource](node.DB).
				Set(ctx, ontology.Resource{ID: alice.OntologyID()})).To(Succeed())
			legacyDB := gorp.Wrap(node.DB.KV(), gorp.WithCodec(msgpack.Codec))
			Expect(gorp.WrapWriter[userv0.Key, userv0.User](legacyDB).Set(ctx, alice)).
				To(Succeed())
			Expect(gorp.WrapWriter[string, authv0.SecureCredentials](legacyDB).Set(
				ctx,
				authv0.SecureCredentials{Username: "alice", Password: hash},
			)).To(Succeed())
			layer := MustOpen(svctestutil.OpenLayer(ctx, node))
			session, _ := MustSucceed2(layer.Auth.Login(ctx, auth.Proof{
				Variant: auth.PasswordProof{
					Credentials: auth.Credentials{Username: "alice", Password: "p"},
				},
			}))
			Expect(session.Subject).To(Equal(alice.OntologyID()))
		},
	)

	// Regression: the composition pass converts data staged by services that open
	// after the panel table (project layouts, task re-key mapping). Running the
	// conversion inside the panel table's own chain silently produced zero panels,
	// because the chain ran before the producers staged anything.
	It(
		"Should adopt a legacy workspace layout as a panel after all tables open",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			db := node.DB
			lpKey := uuid.New().String()
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

			MustOpen(svctestutil.OpenLayer(ctx, node))

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
