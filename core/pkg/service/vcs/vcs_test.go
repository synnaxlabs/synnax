// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package vcs_test

import (
	"bytes"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"
	"uuid"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/vcs"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var ada = vcs.Author{Name: "Ada", Email: "ada@synnax.local"}

var _ = Describe("VCS", func() {
	var (
		node      testutil.Node
		db        *gorp.DB
		layer     *service.Layer
		svc       *vcs.Service
		remoteDir string
		remoteURL string
		proj      project.Project
	)

	openService := func(ctx SpecContext, dir string, cfgs ...vcs.Config) *vcs.Service {
		GinkgoHelper()
		return MustOpen(vcs.OpenService(ctx, append([]vcs.Config{{
			DB:             db,
			Dirname:        dir,
			Channel:        layer.Channel,
			Task:           layer.Task,
			TaskStopper:    layer.TaskStopper,
			Status:         layer.Status,
			Arc:            layer.Arc,
			MatchArcTasks:  layer.ArcTask.MatchArcs,
			Schematic:      layer.Schematic,
			LinePlot:       layer.LinePlot,
			Log:            layer.Log,
			Table:          layer.Table,
			Panel:          layer.Panel,
			Project:        layer.Project,
			Group:          layer.Group,
			FetchInterval:  time.Hour,
			StatusDebounce: 5 * time.Millisecond,
		}}, cfgs...)...))
	}

	tx := func(ctx SpecContext, f func(tx gorp.Tx) error) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, f)).To(Succeed())
	}

	createSchematic := func(ctx SpecContext, name string) schematic.Schematic {
		GinkgoHelper()
		s := schematic.Schematic{Name: name, Project: &proj.Key}
		tx(ctx, func(tx gorp.Tx) error {
			return layer.Schematic.NewWriter(tx).Create(ctx, &s)
		})
		return s
	}

	renameSchematic := func(ctx SpecContext, key uuid.UUID, name string) {
		GinkgoHelper()
		tx(ctx, func(tx gorp.Tx) error {
			_, err := layer.Schematic.NewWriter(tx).Apply(
				ctx,
				key,
				schematic.NewSetAction(schematic.SetPayload{
					Path:  []string{"name"},
					Value: name,
				}),
			)
			return err
		})
	}

	retrieveSchematic := func(ctx SpecContext, key uuid.UUID) schematic.Schematic {
		GinkgoHelper()
		var s schematic.Schematic
		Expect(layer.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(key)).
			Entry(&s).
			Exec(ctx, db)).To(Succeed())
		return s
	}

	connect := func(ctx SpecContext, url string) vcs.Status {
		GinkgoHelper()
		return MustSucceed(svc.Connect(ctx, vcs.ConnectRequest{
			Configuration: "main",
			RemoteURL:     url,
			Author:        ada,
		}))
	}

	commit := func(ctx SpecContext, message string, resources ...string) vcs.Commit {
		GinkgoHelper()
		c, url := MustSucceed2(svc.Commit(ctx, vcs.CommitRequest{
			Message:   message,
			Resources: resources,
			Author:    ada,
		}))
		Expect(url).To(BeEmpty())
		return c
	}

	working := func(ctx SpecContext) []vcs.File {
		GinkgoHelper()
		return MustSucceed(svc.Files(ctx, vcs.RevWorking, ""))
	}

	// pushFromElsewhere commits a rename of the file at from to to on the remote, the
	// way another Core would.
	pushFromElsewhere := func(from, to string, edit func([]byte) []byte) {
		GinkgoHelper()
		dir := GinkgoT().TempDir()
		r := MustSucceed(git.PlainClone(dir, false, &git.CloneOptions{
			URL:           remoteURL,
			ReferenceName: plumbing.NewBranchReferenceName("main"),
		}))
		b := MustSucceed(os.ReadFile(filepath.Join(dir, from)))
		wt := MustSucceed(r.Worktree())
		MustSucceed(wt.Remove(from))
		Expect(os.WriteFile(filepath.Join(dir, to), edit(b), 0o644)).To(Succeed())
		MustSucceed(wt.Add(to))
		MustSucceed(wt.Commit("Edit from another Core", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Bob",
				Email: "bob@synnax.local",
				When:  time.Now(),
			},
		}))
		Expect(r.Push(&git.PushOptions{})).To(Succeed())
	}

	BeforeEach(func(ctx SpecContext) {
		node = testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
		remoteDir = filepath.Join(GinkgoT().TempDir(), "remote.git")
		MustSucceed(git.PlainInit(remoteDir, true))
		remoteURL = "file://" + remoteDir
		svc = openService(ctx, GinkgoT().TempDir())
		proj = project.Project{Name: "Test Stand"}
		tx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})
	})

	Describe("Status", func() {
		It("Should report that version control has not started", func(ctx SpecContext) {
			st := MustSucceed(svc.Status(ctx))
			Expect(st.Enabled).To(BeFalse())
			Expect(st.Changes).To(BeEmpty())
		})

		It(
			"Should report added, modified, and deleted resources",
			func(ctx SpecContext) {
				kept := createSchematic(ctx, "Kept")
				gone := createSchematic(ctx, "Gone")
				connect(ctx, "")
				renameSchematic(ctx, kept.Key, "Renamed")
				tx(ctx, func(tx gorp.Tx) error {
					return layer.Schematic.NewWriter(tx).Delete(ctx, gone.Key)
				})
				added := createSchematic(ctx, "Added")
				Eventually(func(g Gomega) {
					st := MustSucceed(svc.Status(ctx))
					kinds := map[string]vcs.ChangeKind{}
					for _, c := range st.Changes {
						kinds[c.Resource] = c.Kind
					}
					g.Expect(kinds).To(Equal(map[string]vcs.ChangeKind{
						kept.ResourceID().String():  vcs.ChangeModified,
						gone.ResourceID().String():  vcs.ChangeDeleted,
						added.ResourceID().String(): vcs.ChangeAdded,
					}))
				}).Should(Succeed())
			},
		)

		It("Should change the digest of a change on each edit", func(ctx SpecContext) {
			s := createSchematic(ctx, "A")
			connect(ctx, "")
			digest := func(g Gomega) string {
				GinkgoHelper()
				st := MustSucceed(svc.Status(ctx))
				g.Expect(st.Changes).To(HaveLen(1))
				return st.Changes[0].Digest
			}
			renameSchematic(ctx, s.Key, "B")
			var first string
			Eventually(func(g Gomega) { first = digest(g) }).Should(Succeed())
			Expect(first).ToNot(BeEmpty())
			renameSchematic(ctx, s.Key, "C")
			Eventually(func(g Gomega) {
				g.Expect(digest(g)).ToNot(Equal(first))
			}).Should(Succeed())
		})
	})

	Describe("Connect", func() {
		It(
			"Should make the first commit and push it to an empty remote",
			func(ctx SpecContext) {
				createSchematic(ctx, "Valves")
				st := connect(ctx, remoteURL)
				Expect(st.Enabled).To(BeTrue())
				Expect(st.Configuration).To(Equal("main"))
				Expect(st.Head).ToNot(BeNil())
				Expect(st.Head.Message).To(Equal("Start version control"))
				Expect(st.Head.Author).To(Equal("Ada"))
				Expect(st.Changes).To(BeEmpty())
				Expect(st.Waiting).To(BeZero())
				remote := MustSucceed(git.PlainOpen(remoteDir))
				ref := MustSucceed(
					remote.Reference(plumbing.NewBranchReferenceName("main"), true),
				)
				Expect(ref.Hash().String()).To(Equal(st.Head.Hash))
			},
		)

		It(
			"Should keep the stored token when the request has none",
			func(ctx SpecContext) {
				MustSucceed(svc.Connect(ctx, vcs.ConnectRequest{
					Configuration: "main",
					RemoteURL:     remoteURL,
					Token:         "secret",
					Author:        ada,
				}))
				st := connect(ctx, remoteURL)
				Expect(st.TokenSet).To(BeTrue())
			},
		)

		It("Should reject an invalid configuration name", func(ctx SpecContext) {
			Expect(svc.Connect(ctx, vcs.ConnectRequest{Configuration: "../x"})).
				Error().To(MatchError(ContainSubstring("is not a valid name")))
		})
	})

	Describe("Commit", func() {
		It("Should commit only the selected resources", func(ctx SpecContext) {
			connect(ctx, remoteURL)
			a := createSchematic(ctx, "A")
			b := createSchematic(ctx, "B")
			c := commit(ctx, "Add A", a.ResourceID().String())
			Expect(c.Message).To(Equal("Add A"))
			st := MustSucceed(svc.Status(ctx))
			Expect(st.Head.Hash).To(Equal(c.Hash))
			Expect(st.Changes).To(HaveLen(1))
			Expect(st.Changes[0].Resource).To(Equal(b.ResourceID().String()))
			shown := MustSucceed(svc.Show(ctx, c.Hash))
			Expect(shown.Changes).To(HaveLen(1))
			Expect(shown.Changes[0].Resource).To(Equal(a.ResourceID().String()))
			Expect(shown.Changes[0].Kind).To(Equal(vcs.ChangeAdded))
			remote := MustSucceed(git.PlainOpen(remoteDir))
			ref := MustSucceed(
				remote.Reference(plumbing.NewBranchReferenceName("main"), true),
			)
			Expect(ref.Hash().String()).To(Equal(c.Hash))
		})

		It("Should reject a resource with no change", func(ctx SpecContext) {
			s := createSchematic(ctx, "A")
			connect(ctx, "")
			Expect(svc.Commit(ctx, vcs.CommitRequest{
				Message:   "Nothing",
				Resources: []string{s.ResourceID().String()},
				Author:    ada,
			})).Error().To(MatchError(ContainSubstring("has no change")))
		})

		It("Should list commits newest first, by resource", func(ctx SpecContext) {
			connect(ctx, "")
			a := createSchematic(ctx, "A")
			first := commit(ctx, "Add A")
			createSchematic(ctx, "B")
			second := commit(ctx, "Add B")
			all := MustSucceed(svc.Log(ctx, vcs.LogRequest{}))
			Expect(all).To(HaveLen(3))
			Expect(all[0].Hash).To(Equal(second.Hash))
			Expect(all[1].Hash).To(Equal(first.Hash))
			byA := MustSucceed(
				svc.Log(ctx, vcs.LogRequest{Resource: a.ResourceID().String()}),
			)
			Expect(byA).To(HaveLen(1))
			Expect(byA[0].Hash).To(Equal(first.Hash))
			Expect(MustSucceed(svc.Log(ctx, vcs.LogRequest{Limit: 1}))).To(HaveLen(1))
		})
	})

	Describe("Pull request", func() {
		It(
			"Should push a review branch and open a pull request",
			func(ctx SpecContext) {
				var received map[string]any
				var path, auth string
				gh := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						defer GinkgoRecover()
						path, auth = r.URL.Path, r.Header.Get("Authorization")
						Expect(json.UnmarshalRead(r.Body, &received)).To(Succeed())
						w.WriteHeader(http.StatusCreated)
						MustSucceed(
							w.Write(
								[]byte(
									`{"number": 7, "html_url": "https://github.com/x/remote/pull/7"}`,
								),
							),
						)
					}),
				)
				DeferCleanup(gh.Close)
				svc = openService(
					ctx,
					GinkgoT().TempDir(),
					vcs.Config{GitHubAPI: gh.URL},
				)
				MustSucceed(svc.Connect(ctx, vcs.ConnectRequest{
					Configuration: "main",
					RemoteURL:     remoteURL,
					Token:         "secret",
					Author:        ada,
				}))
				s := createSchematic(ctx, "Valves")
				c, url := MustSucceed2(svc.Commit(ctx, vcs.CommitRequest{
					Message:     "Add the valves",
					PullRequest: true,
					Author:      ada,
				}))
				Expect(url).To(Equal("https://github.com/x/remote/pull/7"))
				Expect(path).To(HaveSuffix("/remote/pulls"))
				Expect(auth).To(Equal("Bearer secret"))
				Expect(received).To(HaveKeyWithValue("base", "main"))
				Expect(received).To(HaveKeyWithValue("title", "Add the valves"))
				head := received["head"].(string)
				Expect(head).To(HavePrefix("synnax/add-the-valves-"))
				remote := MustSucceed(git.PlainOpen(remoteDir))
				ref := MustSucceed(
					remote.Reference(plumbing.NewBranchReferenceName(head), true),
				)
				Expect(ref.Hash().String()).To(Equal(c.Hash))
				st := MustSucceed(svc.Status(ctx))
				Expect(st.Head.Hash).ToNot(Equal(c.Hash))
				Expect(st.Changes).To(HaveLen(1))
				Expect(st.Changes[0].Resource).To(Equal(s.ResourceID().String()))
				Expect(st.Changes[0].Review).To(Equal(&vcs.Review{
					Number: 7,
					URL:    "https://github.com/x/remote/pull/7",
				}))
				Expect(MustSucceed(svc.Configurations(ctx))).To(HaveLen(1))
			},
		)
	})

	Describe("Files", func() {
		It(
			"Should read a resource at a commit and at the parent of one",
			func(ctx SpecContext) {
				connect(ctx, "")
				s := createSchematic(ctx, "Valves")
				c := commit(ctx, "Add valves")
				fs := MustSucceed(svc.Files(ctx, c.Hash, s.ResourceID().String()))
				Expect(fs).To(HaveLen(1))
				Expect(
					fs[0].Path,
				).To(Equal("projects/Test Stand/Valves.schematic.json"))
				Expect(fs[0].Content).To(ContainSubstring(`"name": "Valves"`))
				Expect(
					MustSucceed(svc.Files(ctx, c.Hash+"^", s.ResourceID().String())),
				).
					To(BeEmpty())
				root := MustSucceed(svc.Log(ctx, vcs.LogRequest{}))[1]
				Expect(MustSucceed(svc.Files(ctx, root.Hash+"^", ""))).To(BeEmpty())
			},
		)

		It("Should compare two revisions", func(ctx SpecContext) {
			connect(ctx, "")
			s := createSchematic(ctx, "Valves")
			changes := MustSucceed(svc.Compare(ctx, vcs.RevHead, vcs.RevWorking))
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Resource).To(Equal(s.ResourceID().String()))
		})
	})

	Describe("Discard and Write", func() {
		It("Should discard a change and undo the discard", func(ctx SpecContext) {
			s := createSchematic(ctx, "Valves")
			connect(ctx, "")
			renameSchematic(ctx, s.Key, "Pumps")
			inv := MustSucceed(svc.Discard(ctx, nil, svc.DefaultWrite))
			Expect(retrieveSchematic(ctx, s.Key).Name).To(Equal("Valves"))
			Expect(MustSucceed(svc.Status(ctx)).Changes).To(BeEmpty())
			Expect(svc.Write(ctx, inv, svc.DefaultWrite)).To(Succeed())
			Expect(retrieveSchematic(ctx, s.Key).Name).To(Equal("Pumps"))
		})

		It("Should delete an added resource on discard", func(ctx SpecContext) {
			connect(ctx, "")
			s := createSchematic(ctx, "Valves")
			inv := MustSucceed(svc.Discard(ctx, nil, svc.DefaultWrite))
			Expect(inv.Files).To(HaveLen(1))
			Expect(layer.Schematic.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Exec(ctx, db)).To(MatchError(ContainSubstring("not found")))
			Expect(svc.Write(ctx, inv, svc.DefaultWrite)).To(Succeed())
			Expect(retrieveSchematic(ctx, s.Key).Name).To(Equal("Valves"))
		})
	})

	Describe("Restore", func() {
		It("Should restore a resource in place, with its key", func(ctx SpecContext) {
			s := createSchematic(ctx, "Valves")
			first := connect(ctx, "").Head
			renameSchematic(ctx, s.Key, "Pumps")
			commit(ctx, "Rename")
			target := vcs.Target{
				Kind:      vcs.TargetRestore,
				Commit:    first.Hash,
				Resources: []string{s.ResourceID().String()},
			}
			plan := MustSucceed(svc.Plan(ctx, target))
			Expect(plan.Items).To(HaveLen(1))
			Expect(plan.Items[0].Kind).To(Equal(vcs.PlanChange))
			Expect(plan.Items[0].Conflict).To(BeFalse())
			inv := MustSucceed(svc.Apply(ctx, target, nil, svc.DefaultWrite))
			Expect(retrieveSchematic(ctx, s.Key).Name).To(Equal("Valves"))
			st := MustSucceed(svc.Status(ctx))
			Expect(st.Changes).To(HaveLen(1))
			Expect(st.Changes[0].Kind).To(Equal(vcs.ChangeModified))
			Expect(svc.Write(ctx, inv, svc.DefaultWrite)).To(Succeed())
			Expect(retrieveSchematic(ctx, s.Key).Name).To(Equal("Pumps"))
		})
	})

	Describe("Pull", func() {
		It(
			"Should apply incoming commits and resolve a conflict",
			func(ctx SpecContext) {
				s := createSchematic(ctx, "Valves")
				other := createSchematic(ctx, "Pumps")
				connect(ctx, remoteURL)
				pushFromElsewhere(
					"projects/Test Stand/Valves.schematic.json",
					"projects/Test Stand/Remote.schematic.json",
					func(b []byte) []byte {
						return bytes.Replace(
							b,
							[]byte(`"Valves"`),
							[]byte(`"Remote"`),
							1,
						)
					},
				)
				pushFromElsewhere(
					"projects/Test Stand/Pumps.schematic.json",
					"projects/Test Stand/Remote Pumps.schematic.json",
					func(b []byte) []byte {
						return bytes.Replace(
							b,
							[]byte(`"Pumps"`),
							[]byte(`"Remote Pumps"`),
							1,
						)
					},
				)
				renameSchematic(ctx, s.Key, "Mine")
				st := MustSucceed(svc.Fetch(ctx))
				Expect(st.Incoming).To(HaveLen(2))
				Expect(st.IncomingChanges).To(HaveLen(2))
				target := vcs.Target{Kind: vcs.TargetPull}
				plan := MustSucceed(svc.Plan(ctx, target))
				Expect(plan.Items).To(HaveLen(2))
				conflicts := set.New[string]()
				for _, it := range plan.Items {
					if it.Conflict {
						conflicts.Add(it.Resource)
					}
				}
				Expect(conflicts).To(Equal(set.New(s.ResourceID().String())))
				Expect(svc.Apply(ctx, target, nil, svc.DefaultWrite)).
					Error().To(MatchError(vcs.ErrUnresolvedConflict))
				MustSucceed(svc.Apply(ctx, target, []vcs.Choice{{
					Resource: s.ResourceID().String(),
					Choice:   vcs.ChoiceIncoming,
				}}, svc.DefaultWrite))
				Expect(retrieveSchematic(ctx, s.Key).Name).To(Equal("Remote"))
				Expect(retrieveSchematic(ctx, other.Key).Name).To(Equal("Remote Pumps"))
				st = MustSucceed(svc.Status(ctx))
				Expect(st.Incoming).To(BeEmpty())
				Expect(st.Changes).To(BeEmpty())
				Expect(st.Waiting).To(BeZero())
			},
		)

		It("Should keep mine and replay a waiting commit", func(ctx SpecContext) {
			s := createSchematic(ctx, "Valves")
			connect(ctx, remoteURL)
			pushFromElsewhere(
				"projects/Test Stand/Valves.schematic.json",
				"projects/Test Stand/Remote.schematic.json",
				func(b []byte) []byte {
					return bytes.Replace(b, []byte(`"Valves"`), []byte(`"Remote"`), 1)
				},
			)
			renameSchematic(ctx, s.Key, "Mine")
			commit(ctx, "Rename to mine")
			MustSucceed(svc.Fetch(ctx))
			target := vcs.Target{Kind: vcs.TargetPull}
			MustSucceed(svc.Apply(ctx, target, []vcs.Choice{{
				Resource: s.ResourceID().String(),
				Choice:   vcs.ChoiceMine,
			}}, svc.DefaultWrite))
			Expect(retrieveSchematic(ctx, s.Key).Name).To(Equal("Mine"))
			st := MustSucceed(svc.Status(ctx))
			Expect(st.Changes).To(BeEmpty())
			Expect(st.Incoming).To(BeEmpty())
			Expect(st.Waiting).To(BeZero())
			Expect(st.Head.Message).To(Equal("Rename to mine"))
		})
	})

	Describe("Configurations", func() {
		It("Should clone a configuration and load another", func(ctx SpecContext) {
			s := createSchematic(ctx, "Valves")
			connect(ctx, remoteURL)
			st := MustSucceed(svc.Clone(ctx, "stand-2"))
			Expect(st.Configuration).To(Equal("stand-2"))
			renameSchematic(ctx, s.Key, "Stand 2 valves")
			commit(ctx, "Rename for stand 2")
			confs := MustSucceed(svc.Configurations(ctx))
			Expect(confs).To(HaveLen(2))
			Expect(confs[0].Name).To(Equal("main"))
			Expect(confs[0].Current).To(BeFalse())
			Expect(confs[1].Name).To(Equal("stand-2"))
			Expect(confs[1].Current).To(BeTrue())
			target := vcs.Target{Kind: vcs.TargetLoad, Configuration: "main"}
			plan := MustSucceed(svc.Plan(ctx, target))
			Expect(plan.Items).To(HaveLen(1))
			Expect(plan.Items[0].Conflict).To(BeFalse())
			MustSucceed(svc.Apply(ctx, target, nil, svc.DefaultWrite))
			Expect(retrieveSchematic(ctx, s.Key).Name).To(Equal("Valves"))
			st = MustSucceed(svc.Status(ctx))
			Expect(st.Configuration).To(Equal("main"))
			Expect(st.Changes).To(BeEmpty())
			changes := MustSucceed(svc.Compare(
				ctx,
				"configuration/main",
				"configuration/stand-2",
			))
			Expect(changes).To(HaveLen(1))
		})

		It(
			"Should track the previous configuration on undo of a load",
			func(ctx SpecContext) {
				s := createSchematic(ctx, "Valves")
				connect(ctx, "")
				MustSucceed(svc.Clone(ctx, "stand-2"))
				renameSchematic(ctx, s.Key, "Stand 2 valves")
				commit(ctx, "Rename for stand 2")
				target := vcs.Target{Kind: vcs.TargetLoad, Configuration: "main"}
				inv := MustSucceed(svc.Apply(ctx, target, nil, svc.DefaultWrite))
				Expect(inv.Configuration).To(Equal("stand-2"))
				Expect(svc.Write(ctx, inv, svc.DefaultWrite)).To(Succeed())
				Expect(retrieveSchematic(ctx, s.Key).Name).To(Equal("Stand 2 valves"))
				st := MustSucceed(svc.Status(ctx))
				Expect(st.Configuration).To(Equal("stand-2"))
				Expect(st.Changes).To(BeEmpty())
			},
		)

		It(
			"Should reject an undo that tracks a missing configuration",
			func(ctx SpecContext) {
				connect(ctx, "")
				Expect(svc.Write(
					ctx,
					vcs.Inverse{Configuration: "gone"},
					svc.DefaultWrite,
				)).To(MatchError(query.ErrNotFound))
			},
		)

		It("Should reject a clone onto a taken name", func(ctx SpecContext) {
			connect(ctx, "")
			Expect(
				svc.Clone(ctx, "main"),
			).Error().
				To(MatchError(ContainSubstring("exists")))
		})
	})

	Describe("Snapshot", func() {
		It("Should tag the live configuration for a range", func(ctx SpecContext) {
			createSchematic(ctx, "Valves")
			head := connect(ctx, remoteURL).Head
			rng := uuid.New().String()
			clean := MustSucceed(svc.Snapshot(ctx, rng, ada))
			Expect(clean.Commit.Hash).To(Equal(head.Hash))
			Expect(clean.Uncommitted).To(BeZero())
			Expect(clean.Tag).To(HavePrefix("range/" + rng + "/"))
			createSchematic(ctx, "Pumps")
			time.Sleep(2 * time.Millisecond)
			dirty := MustSucceed(svc.Snapshot(ctx, rng, ada))
			Expect(dirty.Commit.Hash).ToNot(Equal(head.Hash))
			Expect(dirty.Uncommitted).To(Equal(1))
			snaps := MustSucceed(svc.Snapshots(ctx, rng))
			Expect(snaps).To(HaveLen(2))
			Expect(snaps[0].Tag).To(Equal(dirty.Tag))
			Expect(snaps[0].Configuration).To(Equal("main"))
			Expect(snaps[0].Uncommitted).To(Equal(1))
			Expect(snaps[1].Tag).To(Equal(clean.Tag))
			Expect(MustSucceed(svc.Status(ctx)).Head.Hash).To(Equal(head.Hash))
			remote := MustSucceed(git.PlainOpen(remoteDir))
			MustSucceed(remote.Reference(plumbing.NewTagReferenceName(dirty.Tag), true))
		})
	})

	Describe("Round trip", func() {
		var (
			idx  channel.Channel
			data channel.Channel
			tsk  task.Task
			auto arc.Arc
		)

		BeforeEach(func(ctx SpecContext) {
			idx = channel.Channel{
				Name:     "vcs_time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			tx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &idx)
			})
			data = channel.Channel{
				Name:       "vcs_pressure",
				DataType:   telem.Float32T,
				LocalIndex: idx.LocalKey,
			}
			tx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &data)
			})
			tsk = task.Task{
				Name:   "Alerts",
				Type:   pagerduty.AlertTaskType,
				Config: msgpack.EncodedJSON{"routing_key": "rk"},
			}
			tx(ctx, func(tx gorp.Tx) error {
				return layer.Task.NewWriter(tx).Create(ctx, &tsk)
			})
			var dirs []directory.Directory
			Expect(layer.Arc.NewDirectoryRetrieve().Entries(&dirs).Exec(ctx, db)).
				To(Succeed())
			var root directory.Key
			for _, d := range dirs {
				if d.Parent == nil {
					root = d.Key
				}
			}
			dir := directory.Directory{Name: "Sequences"}
			tx(ctx, func(tx gorp.Tx) error {
				return layer.Arc.NewWriter(tx).CreateDirectory(ctx, &dir, root)
			})
			auto = arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
				Name: "Press",
				Source: arc.Source{Variant: arc.TextSource{
					Text: text.Text{Raw: "func press() {\n}\n"},
				}},
			}}}
			tx(ctx, func(tx gorp.Tx) error {
				w := layer.Arc.NewWriter(tx)
				if err := w.Create(ctx, &auto); err != nil {
					return err
				}
				return w.Apply(ctx, auto.Base().Key, arc.NewSetDirectoryAction(
					arc.SetDirectoryPayload{Directory: dir.Key},
				))
			})
			g := group.Group{
				Name:   "Propulsion",
				Parent: new(project.ResourceID(proj.Key)),
			}
			tx(ctx, func(tx gorp.Tx) error {
				return layer.Group.NewWriter(tx).Create(ctx, &g)
			})
			tx(ctx, func(tx gorp.Tx) error {
				s := schematic.Schematic{
					Name:    "Valves",
					Project: &proj.Key,
					Group:   &g.Key,
				}
				if err := layer.Schematic.NewWriter(tx).Create(ctx, &s); err != nil {
					return err
				}
				lp := defaults.Fill(lineplot.LinePlot{Name: "Plot", Project: &proj.Key})
				if err := layer.LinePlot.NewWriter(tx).Create(ctx, &lp); err != nil {
					return err
				}
				lg := defaults.Fill(log.Log{Name: "Log", Project: &proj.Key})
				if err := layer.Log.NewWriter(tx).Create(ctx, &lg); err != nil {
					return err
				}
				tb := table.Table{Name: "Table", Project: &proj.Key}
				if err := layer.Table.NewWriter(tx).Create(ctx, &tb); err != nil {
					return err
				}
				pn := panel.Panel{Name: "Panel", Project: &proj.Key}
				if err := layer.Panel.NewWriter(tx).Create(ctx, &pn); err != nil {
					return err
				}
				sym := defaults.Fill(symbol.Symbol{
					Name: "Valve",
					Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
				})
				return layer.Schematic.Symbol.NewWriter(tx).Create(ctx, &sym)
			})
			connect(ctx, "")
		})

		It("Should export every type with no runtime state", func(ctx SpecContext) {
			paths := map[string]string{}
			for _, f := range working(ctx) {
				paths[f.Path] = f.Content
			}
			Expect(paths).To(HaveKey("channels/vcs_time.json"))
			Expect(paths).To(HaveKey("channels/vcs_pressure.json"))
			Expect(paths).To(HaveKey("tasks/Alerts.json"))
			Expect(paths).To(HaveKey("arc/Sequences/_directory.json"))
			Expect(paths).To(HaveKey("arc/Sequences/Press.json"))
			Expect(paths["arc/Sequences/Press.arc"]).To(Equal("func press() {\n}\n"))
			Expect(paths).To(HaveKey("projects/Test Stand/_project.json"))
			Expect(paths).To(HaveKey("projects/Test Stand/Propulsion/_group.json"))
			Expect(
				paths,
			).To(HaveKey("projects/Test Stand/Propulsion/Valves.schematic.json"))
			Expect(paths).To(HaveKey("projects/Test Stand/Plot.lineplot.json"))
			Expect(paths).To(HaveKey("projects/Test Stand/Log.log.json"))
			Expect(paths).To(HaveKey("projects/Test Stand/Table.table.json"))
			Expect(paths).To(HaveKey("projects/Test Stand/Panel.panel.json"))
			Expect(paths).To(HaveKey("symbols/Valve.json"))
			Expect(paths["tasks/Alerts.json"]).To(ContainSubstring(`"rack"`))
			Expect(paths["tasks/Alerts.json"]).ToNot(ContainSubstring(`"status":`))
			Expect(paths["arc/Sequences/Press.json"]).ToNot(ContainSubstring(`"text":`))
			Expect(
				paths["arc/Sequences/Press.json"],
			).ToNot(ContainSubstring(`"program":`))
		})

		It("Should export the same bytes twice", func(ctx SpecContext) {
			Expect(working(ctx)).To(Equal(working(ctx)))
		})

		It(
			"Should export the same bytes after an apply of every file",
			func(ctx SpecContext) {
				before := working(ctx)
				Expect(
					svc.Write(ctx, vcs.Inverse{Files: before}, svc.DefaultWrite),
				).To(Succeed())
				Expect(working(ctx)).To(Equal(before))
			},
		)

		It(
			"Should create every deleted resource again from the head",
			func(ctx SpecContext) {
				head := MustSucceed(svc.Status(ctx)).Head
				before := working(ctx)
				tx(ctx, func(tx gorp.Tx) error {
					if err := layer.Task.NewWriter(tx).
						Delete(ctx, tsk.Key, false); err != nil {
						return err
					}
					return layer.Arc.NewWriter(tx).Delete(ctx, auto.Base().Key)
				})
				Eventually(func(g Gomega) {
					g.Expect(MustSucceed(svc.Status(ctx)).Changes).To(HaveLen(2))
				}).Should(Succeed())
				inv := MustSucceed(svc.Discard(ctx, nil, svc.DefaultWrite))
				Expect(inv.Files).To(BeEmpty())
				Expect(inv.Removed).To(ConsistOf(
					task.ResourceID(tsk.Key).String(),
					auto.ResourceID().String(),
				))
				Expect(working(ctx)).To(Equal(before))
				target := vcs.Target{Kind: vcs.TargetRestore, Commit: head.Hash}
				Expect(MustSucceed(svc.Plan(ctx, target)).Items).To(BeEmpty())
			},
		)

		It("Should update the source of an Arc in place", func(ctx SpecContext) {
			fs := working(ctx)
			for i := range fs {
				if fs[i].Path == "arc/Sequences/Press.arc" {
					fs[i].Content = "func press() {\n    wait()\n}\n"
				}
			}
			Expect(
				svc.Write(ctx, vcs.Inverse{Files: fs}, svc.DefaultWrite),
			).To(Succeed())
			var a arc.Arc
			Expect(layer.Arc.NewRetrieve().
				Where(arc.MatchKeys(auto.Base().Key)).
				Entry(&a).
				Exec(ctx, db)).To(Succeed())
			src := a.Base().Source.Variant.(arc.TextSource)
			Expect(
				src.Text.Materialize().Raw,
			).To(Equal("func press() {\n    wait()\n}\n"))
			st := MustSucceed(svc.Status(ctx))
			Expect(st.Changes).To(HaveLen(1))
			Expect(strings.HasPrefix(st.Changes[0].Resource, "arc:")).To(BeTrue())
		})
	})
})
