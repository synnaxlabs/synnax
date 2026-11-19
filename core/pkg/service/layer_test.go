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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/file"
	sectestutil "github.com/synnaxlabs/synnax/pkg/security/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	authv0 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	groupv0 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	lineplotversions "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	lineplotv0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	logv0 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	svcnode "github.com/synnaxlabs/synnax/pkg/service/node"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	pagerdutyv1 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v1"
	pagerdutyv2 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	panelversions "github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	panelv0 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	projectv0 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	schematicv0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	tablev0 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskversions "github.com/synnaxlabs/synnax/pkg/service/task/versions"
	taskv0 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v0"
	taskv2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	taskv3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	userv0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/msgpack"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"golang.org/x/crypto/bcrypt"
)

var _ = Describe("Layer", func() {
	It("Should report the expiry of the Core's CA and listener certificates",
		func(ctx SpecContext) {
			certFS := xfs.NewMem()
			sectestutil.GenerateCerts(certFS)
			certsDir := MustSucceed(certFS.Sub(cert.DefaultLoaderConfig.CertsDir))
			n := testutil.NewNode(ctx)
			layer := MustOpen(svctestutil.OpenLayer(ctx, n, service.LayerConfig{
				CA: MustSucceed(cert.NewLoader(cert.LoaderConfig{FS: certFS})),
				Listeners: []certificate.Listener{{
					Address: "localhost:9090",
					Source: MustSucceed(
						file.NewSource(certsDir, "node.crt", "node.key"),
					),
				}},
			}))
			prefix := svcnode.ResourceID(n.Identity.HostKey).String() + ":certificate:"
			Eventually(func(g Gomega) {
				var found []status.Status[any]
				g.Expect(layer.Status.NewRetrieve[any]().
					WherePrefix(prefix).
					Entries(&found).
					Exec(ctx, n.DB)).To(Succeed())
				g.Expect(lo.Map(found, func(s status.Status[any], _ int) string {
					return s.Name
				})).To(ConsistOf(
					HavePrefix("CA certificate "),
					Equal("Certificate of localhost:9090"),
				))
			}).Should(Succeed())
		})

	It("Should cap external channels at the license's limit", func(ctx SpecContext) {
		signer := svctestutil.NewSigner()
		lic := svctestutil.NewLicense()
		lic.Channels = 50
		node := testutil.NewNode(ctx)
		layer := MustOpen(
			svctestutil.OpenLayer(ctx, node, service.LayerConfig{
				License: license.ServiceConfig{
					Key:     signer.Sign(lic),
					Anchors: signer.Anchors,
				},
			}),
		)
		var existing []channel.Channel
		Expect(layer.Channel.NewRetrieve().Entries(&existing).Exec(ctx, node.DB)).
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
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).
				CreateMany(ctx, new(newIndices(50-external)))
		})).To(Succeed())
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).CreateMany(ctx, new(newIndices(1)))
		})).To(And(
			MatchError(license.ErrTooMany),
			MatchError(ContainSubstring("limit is 50 channels")),
		))
	})

	DescribeTable(
		"Should publish NI resource changes on signal channels",
		func(
			ctx SpecContext,
			name string,
			create func(context.Context, gorp.Tx, *service.Layer) uuid.UUID,
			remove func(context.Context, gorp.Tx, *service.Layer, uuid.UUID) error,
		) {
			node := testutil.NewNode(ctx)
			layer := MustOpen(svctestutil.OpenLayer(ctx, node))
			openStreamer := func(
				name string,
			) confluence.Outlet[framer.StreamerResponse] {
				GinkgoHelper()
				var ch channel.Channel
				Expect(layer.Channel.NewRetrieve().
					Where(channel.MatchNames(name)).
					Entry(&ch).
					Exec(ctx, node.DB)).To(Succeed())
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

			var key uuid.UUID
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				key = create(ctx, tx, layer)
				return nil
			})).To(Succeed())
			var res framer.StreamerResponse
			Eventually(sets.Outlet(), 5*time.Second).Should(Receive(&res))
			var written struct {
				Key uuid.UUID `json:"key"`
			}
			for sample := range res.Frame.SeriesAt(0).Samples() {
				Expect(json.Unmarshal(sample, &written)).To(Succeed())
			}
			Expect(written.Key).To(Equal(key))

			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return remove(ctx, tx, layer, key)
			})).To(Succeed())
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
			func(ctx context.Context, tx gorp.Tx, layer *service.Layer) uuid.UUID {
				GinkgoHelper()
				c := chassis.Chassis{
					Rack:         layer.Rack.EmbeddedKey,
					Name:         "PXIe-1071",
					ResourceName: "Mod1",
					Model:        "PXIe-1071",
				}
				Expect(layer.NI.Chassis.NewWriter(tx).Create(ctx, &c)).To(Succeed())
				return c.Key
			},
			func(
				ctx context.Context,
				tx gorp.Tx,
				layer *service.Layer,
				key uuid.UUID,
			) error {
				return layer.NI.Chassis.NewWriter(tx).Delete(ctx, key)
			},
		),
		Entry(
			"device",
			"device",
			func(ctx context.Context, tx gorp.Tx, layer *service.Layer) uuid.UUID {
				GinkgoHelper()
				d := device.Device{
					Rack:         layer.Rack.EmbeddedKey,
					Name:         "PXI-6255",
					ResourceName: "Mod1",
					Model:        "PXI-6255",
				}
				Expect(layer.NI.Devices.NewWriter(tx).Create(ctx, &d)).To(Succeed())
				return d.Key
			},
			func(
				ctx context.Context,
				tx gorp.Tx,
				layer *service.Layer,
				key uuid.UUID,
			) error {
				return layer.NI.Devices.NewWriter(tx).Delete(ctx, key)
			},
		),
	)

	It("Should refuse a framer writer on a change channel", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		layer := MustOpen(svctestutil.OpenLayer(ctx, node))
		var ch channel.Channel
		Expect(layer.Channel.NewRetrieve().
			Where(channel.MatchNames("sy_range_set")).
			Entry(&ch).
			Exec(ctx, node.DB)).To(Succeed())
		Expect(layer.Framer.NewStreamWriter(ctx, framer.WriterConfig{
			Keys:  channel.Keys{ch.Key()},
			Start: telem.SecondTS,
		})).Error().To(MatchError(fmt.Sprintf(
			"channel %d holds changes that only the Core writes: validation error",
			ch.Key(),
		)))
	})

	It("Should delete a stored snapshot task", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		legacyDB := gorp.Wrap(node.DB.KV(), gorp.WithCodec(msgpack.Codec))
		stored := []taskv0.Task{
			{
				Key:      taskv0.Key(uint64(65537)<<32 | 1),
				Name:     "Snapshot",
				Type:     pagerduty.AlertTaskType,
				Config:   msgpack.EncodedJSON{"routing_key": "rk"},
				Snapshot: true,
			},
			{
				Key:    taskv0.Key(uint64(65537)<<32 | 2),
				Name:   "Live",
				Type:   pagerduty.AlertTaskType,
				Config: msgpack.EncodedJSON{"routing_key": "rk"},
			},
		}
		for _, t := range stored {
			Expect(gorp.WrapWriter[taskv0.Key, taskv0.Task](legacyDB).Set(ctx, t)).
				To(Succeed())
		}
		layer := MustOpen(svctestutil.OpenLayer(ctx, node))
		Expect(layer.Task.NewRetrieve().
			Where(task.MatchNames("Snapshot")).
			Exists(ctx, node.DB)).To(BeFalse())
		Expect(layer.Task.NewRetrieve().
			Where(task.MatchNames("Live")).
			Exists(ctx, node.DB)).To(BeTrue())
	})

	// The snapshot gets no secret, because the task chain deletes it before the secret
	// step runs.
	DescribeTable(
		"Should move a stored PagerDuty routing key into a secret after the task chain",
		func(
			ctx SpecContext,
			lastApplied string,
			storeTask func(context.Context, *gorp.DB, uuid.UUID, string, bool) error,
		) {
			node := testutil.NewNode(ctx)
			db := node.DB
			chain := lo.Map(
				taskversions.NewMigrations(taskversions.MigrationsConfig{}),
				func(m gorp.Migration, _ int) string { return m.Key() },
			)
			Expect(chain).To(ContainElement(lastApplied))
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:        db,
				Namespace: "Task",
				Migrations: lo.Map(
					chain[:slices.Index(chain, lastApplied)+1],
					func(key string, _ int) gorp.Migration {
						return migrate.New(
							key,
							func(context.Context, gorp.Tx, alamos.Instrumentation) error {
								return nil
							},
						)
					},
				),
			})).To(Succeed())
			store := func(name string, snapshot bool) uuid.UUID {
				GinkgoHelper()
				key := uuid.New()
				Expect(storeTask(ctx, db, key, name, snapshot)).To(Succeed())
				Expect(gorp.WrapWriter[uuid.UUID, pagerdutyv1.TaskConfig](db).Set(
					ctx,
					pagerdutyv1.TaskConfig{Key: key, RoutingKey: "rk"},
				)).To(Succeed())
				return key
			}
			live := store("Live", false)
			snapshot := store("Snapshot", true)
			layer := MustOpen(svctestutil.OpenLayer(ctx, node))
			var secrets []secret.Secret
			Expect(layer.Secret.NewRetrieve().Entries(&secrets).Exec(ctx, db)).
				To(Succeed())
			Expect(secrets).To(HaveLen(1))
			Expect(secrets[0].Base().Name).To(Equal("Live routing key"))
			key := secrets[0].Base().Key
			Expect(layer.Secret.Reveal(ctx, db, key)).
				To(Equal(map[secret.Key][]byte{key: []byte("rk")}))
			data := MustSucceed(layer.PagerDuty.Alert.Read(ctx, db, live))
			Expect(data).To(HaveKeyWithValue("routing_key", key.String()))
			var upgraded task.Task
			Expect(layer.Task.NewRetrieve().
				Where(task.MatchKeys(live)).
				Entry(&upgraded).
				Exec(ctx, db)).To(Succeed())
			Expect(upgraded.ConfigHash).
				To(Equal(MustSucceed(layer.PagerDuty.Alert.Hash(ctx, db, live))))
			for _, k := range []uuid.UUID{live, snapshot} {
				Expect(db.Get(pagerdutyv2.PlainRoutingKeyKVKey(k))).Error().
					To(MatchError(query.ErrNotFound))
			}
		},
		Entry(
			"from Core v0.60, which stored task v2",
			"v60_require_name",
			func(
				ctx context.Context,
				db *gorp.DB,
				key uuid.UUID,
				name string,
				snapshot bool,
			) error {
				return gorp.WrapWriter[taskv2.Key, taskv2.Task](
					db,
				).Set(ctx, taskv2.Task{
					Key:      key,
					Name:     name,
					Type:     pagerduty.AlertTaskType,
					Snapshot: snapshot,
				})
			},
		),
		Entry(
			"from Core v0.61, which stored task v3",
			"v61_task_range",
			func(
				ctx context.Context,
				db *gorp.DB,
				key uuid.UUID,
				name string,
				snapshot bool,
			) error {
				t := taskv3.Task{
					Key:      key,
					Name:     name,
					Type:     pagerduty.AlertTaskType,
					Snapshot: snapshot,
				}
				if snapshot {
					t.Range = new(uuid.New())
				}
				return gorp.WrapWriter[taskv3.Key, taskv3.Task](db).Set(ctx, t)
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
				Set(ctx, ontology.Resource{ID: alice.ResourceID()})).To(Succeed())
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
			Expect(session.Subject).To(Equal(alice.ResourceID()))
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
			plot := defaults.Fill(lineplot.LinePlot{Key: uuid.New(), Name: "Pressures"})
			lpKey := plot.Key.String()
			lpID := resource.ID{Type: resource.TypeLineplot, Key: lpKey}
			Expect(gorp.WrapWriter[string, ontology.Resource](db).
				Set(ctx, ontology.Resource{ID: lpID})).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "LinePlot",
				Migrations: lineplotversions.Migrations,
			})).To(Succeed())
			Expect(gorp.NewCreate[lineplot.Key, lineplot.LinePlot]().
				Entry(&plot).
				Exec(ctx, db)).To(Succeed())
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
			wsID := resource.ID{Type: "workspace", Key: ws.Key.String()}
			Expect(gorp.WrapWriter[string, ontology.Resource](db).
				Set(ctx, ontology.Resource{ID: wsID})).To(Succeed())

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

			By("Giving the panel to the migrated project")
			Expect(panels[0].Project).To(HaveValue(Equal(ws.Key)))
			rel := ontology.Relationship{
				From: resource.ID{Type: resource.TypeProject, Key: ws.Key.String()},
				Type: ontology.RelationshipTypeParentOf,
				To:   panels[0].ResourceID(),
			}
			Expect(MustSucceed(gorp.NewRetrieve[string, ontology.Relationship]().
				Where(gorp.MatchKeys[string, ontology.Relationship](rel.GorpKey())).
				Exists(ctx, db))).To(BeFalse())
		},
	)

	It(
		"Should delete a group on a v0.58 Core that stores panels",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			tab := func(t resource.Type) panelv0.Tab {
				return panelv0.Tab{Variant: panelv0.ResourceTab{
					Key:      uuid.New(),
					Resource: resource.ID{Type: t, Key: uuid.New().String()},
				}}
			}
			Expect(gorp.WrapWriter[panelv0.Key, panelv0.Panel](node.DB).Set(
				ctx,
				panelv0.Panel{
					Key:  uuid.New(),
					Name: "Main",
					Root: panelv0.Node{Variant: panelv0.SplitNode{
						Direction: spatial.DirectionX,
						Size:      0.5,
						First: panelv0.Node{
							Variant: panelv0.LeafNode{Tabs: []panelv0.Tab{
								{Variant: panelv0.ViewTab{
									Key:  uuid.New(),
									Type: "docs",
									Args: msgpack.EncodedJSON{"page": "intro"},
								}},
								tab(resource.TypeLog),
							}},
						},
						Last: panelv0.Node{Variant: panelv0.LeafNode{
							Tabs: []panelv0.Tab{tab(resource.TypeTable)},
						}},
					}},
				},
			)).To(Succeed())
			layer := MustOpen(svctestutil.OpenLayer(ctx, node))
			g := group.Group{Name: "Valves", Scope: resource.TypeChannel}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Group.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Group.NewWriter(tx).Delete(ctx, g.Key)
			})).To(Succeed())
		},
	)
})

var _ = Describe("Workspace-era upgrade", func() {
	It(
		"Should keep every visualization in the project its workspace becomes",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			legacyDB := gorp.Wrap(node.DB.KV(), gorp.WithCodec(msgpack.Codec))
			ws := projectv0.Workspace{Key: uuid.New(), Name: "Workspace"}
			Expect(gorp.WrapWriter[projectv0.Key, projectv0.Workspace](legacyDB).
				Set(ctx, ws)).To(Succeed())
			g := groupv0.Group{Key: uuid.New(), Name: "Group"}
			Expect(gorp.WrapWriter[groupv0.Key, groupv0.Group](legacyDB).Set(ctx, g)).
				To(Succeed())
			loose := schematicv0.Schematic{Key: uuid.New(), Name: "Loose"}
			grouped := schematicv0.Schematic{Key: uuid.New(), Name: "Grouped"}
			Expect(gorp.WrapWriter[schematicv0.Key, schematicv0.Schematic](legacyDB).
				Set(ctx, loose, grouped)).To(Succeed())
			lp := lineplotv0.LinePlot{Key: uuid.New(), Name: "Line Plot"}
			Expect(gorp.WrapWriter[lineplotv0.Key, lineplotv0.LinePlot](legacyDB).
				Set(ctx, lp)).To(Succeed())
			lg := logv0.Log{Key: uuid.New(), Name: "Log"}
			Expect(gorp.WrapWriter[logv0.Key, logv0.Log](legacyDB).Set(ctx, lg)).
				To(Succeed())
			tbl := tablev0.Table{Key: uuid.New(), Name: "Table"}
			Expect(gorp.WrapWriter[tablev0.Key, tablev0.Table](legacyDB).Set(ctx, tbl)).
				To(Succeed())
			var (
				wsID      = resource.ID{Type: "workspace", Key: ws.Key.String()}
				groupID   = resource.ID{Type: resource.TypeGroup, Key: g.Key.String()}
				groupedID = schematic.ResourceID(grouped.Key)
				owned     = []resource.ID{
					groupID,
					schematic.ResourceID(loose.Key),
					lineplot.ResourceID(lp.Key),
					log.ResourceID(lg.Key),
					table.ResourceID(tbl.Key),
				}
				resources = append(
					make([]ontology.Resource, 0, 2+len(owned)),
					ontology.Resource{ID: wsID},
					ontology.Resource{ID: groupedID},
				)
				edges = append(
					make([]ontology.Relationship, 0, 1+len(owned)),
					ontology.Relationship{
						From: groupID,
						Type: ontology.RelationshipTypeParentOf,
						To:   groupedID,
					},
				)
			)
			for _, id := range owned {
				resources = append(resources, ontology.Resource{ID: id})
				edges = append(edges, ontology.Relationship{
					From: wsID,
					Type: ontology.RelationshipTypeParentOf,
					To:   id,
				})
			}
			Expect(gorp.WrapWriter[string, ontology.Resource](node.DB).
				Set(ctx, resources...)).To(Succeed())
			Expect(gorp.WrapWriter[string, ontology.Relationship](node.DB).
				Set(ctx, edges...)).To(Succeed())

			layer := MustOpen(svctestutil.OpenLayer(ctx, node))

			var schematics []schematic.Schematic
			Expect(layer.Schematic.NewRetrieve().
				Where(schematic.MatchKeys(loose.Key, grouped.Key)).
				Entries(&schematics).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(schematics).To(HaveLen(2))
			for _, s := range schematics {
				Expect(s.Project).To(HaveValue(Equal(ws.Key)))
			}
			var plot lineplot.LinePlot
			Expect(layer.LinePlot.NewRetrieve().
				Where(lineplot.MatchKeys(lp.Key)).
				Entry(&plot).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(plot.Project).To(HaveValue(Equal(ws.Key)))
			var l log.Log
			Expect(layer.Log.NewRetrieve().
				Where(log.MatchKeys(lg.Key)).
				Entry(&l).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(l.Project).To(HaveValue(Equal(ws.Key)))
			var t table.Table
			Expect(layer.Table.NewRetrieve().
				Where(table.MatchKeys(tbl.Key)).
				Entry(&t).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(t.Project).To(HaveValue(Equal(ws.Key)))
		},
	)
})

var _ = Describe("Resource search", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
	})

	DescribeTable(
		"Should search every type that names records",
		func(ctx SpecContext, t resource.Type) {
			MustSucceed(layer.Resource.Search(ctx, db, resource.SearchQuery{
				Term:  "pump",
				Types: []resource.Type{t},
			}))
		},
		Entry("Arc", resource.TypeArc),
		Entry("Channel", resource.TypeChannel),
		Entry("Color", resource.TypeColor),
		Entry("Device", resource.TypeDevice),
		Entry("Enumeration", resource.TypeEnumeration),
		Entry("Group", resource.TypeGroup),
		Entry("Label", resource.TypeLabel),
		Entry("Lineplot", resource.TypeLineplot),
		Entry("Log", resource.TypeLog),
		Entry("NiChassis", resource.TypeNiChassis),
		Entry("NiDevice", resource.TypeNiDevice),
		Entry("Node", resource.TypeNode),
		Entry("Panel", resource.TypePanel),
		Entry("Policy", resource.TypePolicy),
		Entry("Project", resource.TypeProject),
		Entry("Rack", resource.TypeRack),
		Entry("Range", resource.TypeRange),
		Entry("Role", resource.TypeRole),
		Entry("Schematic", resource.TypeSchematic),
		Entry("SchematicSymbol", resource.TypeSchematicSymbol),
		Entry("Status", resource.TypeStatus),
		Entry("Table", resource.TypeTable),
		Entry("Task", resource.TypeTask),
		Entry("User", resource.TypeUser),
		Entry("View", resource.TypeView),
	)

	It("Should find NI chassis and devices by name", func(ctx SpecContext) {
		c := chassis.Chassis{
			Rack:         layer.Rack.EmbeddedKey,
			Name:         "Pump Chassis",
			ResourceName: "PXI1",
			Model:        "PXIe-1071",
		}
		d := device.Device{
			Rack:         layer.Rack.EmbeddedKey,
			Name:         "Pump Module",
			ResourceName: "Mod1",
			Model:        "NI 9205",
			Chassis:      &c.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.NI.Chassis.NewWriter(tx).Create(ctx, &c); err != nil {
				return err
			}
			return layer.NI.Devices.NewWriter(tx).Create(ctx, &d)
		})).To(Succeed())
		res := MustSucceed(layer.Resource.Search(ctx, db, resource.SearchQuery{
			Term:  "pump",
			Types: []resource.Type{resource.TypeNiChassis, resource.TypeNiDevice},
		}))
		Expect(lo.Map(res, func(r resource.Resource, _ int) resource.ID {
			return r.ID
		})).To(ConsistOf(c.ResourceID(), d.ResourceID()))
	})
})

var _ = Describe("Group moves", Ordered, func() {
	var (
		l      *service.Layer
		db     *gorp.DB
		src    project.Project
		dest   project.Project
		g      group.Group
		owners func(ctx SpecContext) []*project.Key
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		l = MustOpen(svctestutil.OpenLayer(ctx, node))
		src = project.Project{Name: "Source"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Project.NewWriter(tx).Create(ctx, &src)
		})).To(Succeed())
		dest = project.Project{Name: "Destination"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Project.NewWriter(tx).Create(ctx, &dest)
		})).To(Succeed())
		g = group.Group{Name: "Moved", Parent: new(project.ResourceID(src.Key))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		sch := schematic.Schematic{Name: "Schematic", Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
		lp := defaults.Fill(lineplot.LinePlot{Name: "Plot", Group: &g.Key})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.LinePlot.NewWriter(tx).Create(ctx, &lp)
		})).To(Succeed())
		lg := defaults.Fill(log.Log{Name: "Log", Group: &g.Key})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Log.NewWriter(tx).Create(ctx, &lg)
		})).To(Succeed())
		tb := table.Table{Name: "Table", Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Table.NewWriter(tx).Create(ctx, &tb)
		})).To(Succeed())
		pn := panel.Panel{Name: "Panel", Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Panel.NewWriter(tx).Create(ctx, &pn)
		})).To(Succeed())
		owners = func(ctx SpecContext) []*project.Key {
			GinkgoHelper()
			Expect(l.Schematic.NewRetrieve().
				Where(schematic.MatchKeys(sch.Key)).Entry(&sch).Exec(ctx, db)).
				To(Succeed())
			Expect(l.LinePlot.NewRetrieve().
				Where(lineplot.MatchKeys(lp.Key)).Entry(&lp).Exec(ctx, db)).
				To(Succeed())
			Expect(l.Log.NewRetrieve().
				Where(log.MatchKeys(lg.Key)).Entry(&lg).Exec(ctx, db)).
				To(Succeed())
			Expect(l.Table.NewRetrieve().
				Where(table.MatchKeys(tb.Key)).Entry(&tb).Exec(ctx, db)).
				To(Succeed())
			Expect(l.Panel.NewRetrieve().
				Where(panel.MatchKeys(pn.Key)).Entry(&pn).Exec(ctx, db)).
				To(Succeed())
			return []*project.Key{
				sch.Project,
				lp.Project,
				lg.Project,
				tb.Project,
				pn.Project,
			}
		}
	})

	It(
		"Should create each visualization in the project of its group",
		func(ctx SpecContext) { Expect(owners(ctx)).To(HaveEach(Equal(&src.Key))) },
	)

	It(
		"Should move the visualizations of a group into the project it moves under",
		func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return l.Group.NewWriter(tx).
					SetParent(ctx, new(project.ResourceID(dest.Key)), g.Key)
			})).To(Succeed())
			Expect(owners(ctx)).To(HaveEach(Equal(&dest.Key)))
		},
	)

	It(
		"Should reject a move of a group that holds visualizations to the top level",
		func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return l.Group.NewWriter(tx).SetParent(ctx, nil, g.Key)
			})).To(MatchError(
				"group: a schematic sits only in a group under a project: validation error",
			))
			Expect(owners(ctx)).To(HaveEach(Equal(&dest.Key)))
		},
	)
})

var _ = Describe("LayerConfig", func() {
	It("Should read a file secret of the Core's rack from its secrets folder", func(
		ctx SpecContext,
	) {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "token"), []byte("t"), 0o600)).
			To(Succeed())
		node := testutil.NewNode(ctx)
		layer := MustOpen(svctestutil.OpenLayer(
			ctx,
			node,
			service.LayerConfig{SecretsDir: dir},
		))
		// The Core's rack runs the integrations of the Core, such as Sift.
		var coreRack rack.Rack
		Expect(layer.Rack.NewRetrieve().
			Where(rack.MatchIntegration(sift.Make)).
			Entry(&coreRack).
			Exec(ctx, node.DB)).To(Succeed())
		s := secret.Secret{Variant: secret.FileSecret{
			Base:     secret.Base{Name: "Token"},
			FileName: "token",
			Rack:     coreRack.Key,
		}}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Secret.NewWriter(tx).Set(ctx, &s, nil)
		})).To(Succeed())
		key := s.Base().Key
		Expect(layer.Secret.Reveal(ctx, node.DB, key)).
			To(Equal(map[secret.Key][]byte{key: []byte("t")}))
	})

	It("Should give the Core's rack a file browser task", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		layer := MustOpen(svctestutil.OpenLayer(ctx, node))
		var coreRack rack.Rack
		Expect(layer.Rack.NewRetrieve().
			Where(rack.MatchIntegration("file")).
			Entry(&coreRack).
			Exec(ctx, node.DB)).To(Succeed())
		var browser task.Task
		Expect(layer.Task.NewRetrieve().
			Where(task.And(
				task.MatchRacks(coreRack.Key),
				task.MatchTypes("file_browser"),
			)).
			Entry(&browser).
			Exec(ctx, node.DB)).To(Succeed())
		Expect(browser.Name).To(Equal("File Browser"))
		Expect(browser.Internal).To(BeTrue())
	})

	It("Should refuse a secrets folder that does not exist", func(ctx SpecContext) {
		dir := filepath.Join(GinkgoT().TempDir(), "missing")
		Expect(svctestutil.OpenLayer(
			ctx,
			testutil.NewNode(ctx),
			service.LayerConfig{SecretsDir: dir},
		)).Error().To(SatisfyAll(
			MatchError(os.ErrNotExist),
			MatchError(HavePrefix("secrets folder: ")),
			MatchError(ContainSubstring(dir)),
		))
	})

	It("Should require a secrets key", func() {
		cfg := service.LayerConfig{
			Distribution: &distribution.Layer{},
			Security: MustSucceed(security.NewProvider(security.ProviderConfig{
				Insecure: new(true),
			})),
		}
		Expect(cfg.Validate()).To(MatchError(ContainSubstring("secrets_key: required")))
	})

	It("Should reject an unknown auth method", func() {
		cfg := service.LayerConfig{
			AuthMethods: []auth.Method{auth.MethodPassword, "carrier_pigeon"},
		}
		Expect(cfg.Validate()).To(MatchError(And(
			ContainSubstring("auth_methods"),
			ContainSubstring("carrier_pigeon"),
		)))
	})
})

var _ = Describe("Auth methods", func() {
	DescribeTable(
		"Should accept only the listed methods",
		func(ctx SpecContext, listed, accepted []auth.Method) {
			node := testutil.NewNode(ctx)
			layer := MustOpen(svctestutil.OpenLayer(ctx, node, service.LayerConfig{
				AuthMethods: listed,
			}))
			Expect(layer.Auth.Methods()).To(Equal(accepted))
		},
		Entry(
			"password only",
			[]auth.Method{auth.MethodPassword},
			[]auth.Method{auth.MethodPassword},
		),
		Entry(
			"API key only",
			[]auth.Method{auth.MethodAPIKey},
			[]auth.Method{auth.MethodAPIKey},
		),
		Entry(
			"every method for an empty list",
			[]auth.Method{},
			[]auth.Method{auth.MethodAPIKey, auth.MethodPassword},
		),
		Entry(
			"no OIDC without a provider",
			[]auth.Method{auth.MethodPassword, auth.MethodOIDC},
			[]auth.Method{auth.MethodPassword},
		),
	)
	It("Should accept OIDC with a provider", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		layer := MustOpen(svctestutil.OpenLayer(ctx, node, service.LayerConfig{
			OIDCProviders: []oidc.Provider{{
				Name:     "idp",
				Issuer:   "https://idp.example.com",
				ClientID: "synnax",
			}},
		}))
		Expect(layer.Auth.Methods()).To(Equal([]auth.Method{
			auth.MethodAPIKey,
			auth.MethodOIDC,
			auth.MethodPassword,
		}))
	})
	It("Should reject OIDC as the only method without a provider", func(
		ctx SpecContext,
	) {
		node := testutil.NewNode(ctx)
		Expect(svctestutil.OpenLayer(ctx, node, service.LayerConfig{
			AuthMethods: []auth.Method{auth.MethodOIDC},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("oidc needs at least one provider")),
		))
	})
})
