// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/arc/drift"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/crdt"
	. "github.com/synnaxlabs/x/lsp/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/uri"
)

var _ = Describe("Change propagation", Ordered, func() {
	const (
		moduleV1 = "export func f() {t u8} i64 { return 1 }"
		moduleV2 = "export func f() {t u8} i64 { return 2 }"
		importer = "import pt_m\n\npt_trig -> pt_m.f() -> pt_out"
	)
	var root directory.Key
	// write runs fn in a transaction of its own, so its commit schedules resyncs.
	write := func(ctx context.Context, fn func(arc.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error { return fn(svc.NewWriter(tx)) })
	}
	textSource := func(raw string) arc.Source {
		return arc.Source{Variant: arc.TextSource{Text: text.Text{Raw: raw}}}
	}
	createArc := func(ctx SpecContext, t arc.Type, name, raw string) arc.Arc {
		GinkgoHelper()
		a := newArc(t, arc.Base{Name: name, Source: textSource(raw)})
		Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &a) })).
			To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(write(ctx, func(w arc.Writer) error {
				return w.Delete(ctx, a.Base().Key)
			})).To(Succeed())
		})
		return a
	}
	// editor is the last replica that overwrite used. Each call takes a new one, so its
	// character ids never collide with an earlier call's.
	var editor uint32 = 100
	// overwrite dispatches the edits that replace the text of a with raw and returns
	// the Arc as stored.
	overwrite := func(ctx SpecContext, a arc.Arc, raw string) arc.Arc {
		GinkgoHelper()
		key := a.Base().Key
		var stored arc.Arc
		Expect(svc.NewRetrieve().
			Where(arc.MatchKeys(key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		doc := stored.Base().Source.Variant.(arc.TextSource).Text.Doc
		editor++
		client := crdt.New(editor)
		client.Load(doc.Inserts, doc.Deletes)
		var acts []arc.Action
		for _, op := range client.Delete(0, client.Len()) {
			del := arc.DeleteCharPayload{ID: op.ID}
			acts = append(acts, arc.NewDeleteCharAction(del))
		}
		for _, op := range client.Insert(0, raw) {
			acts = append(acts, arc.NewInsertCharAction(arc.InsertCharPayload{
				ID: op.ID, Origin: op.Origin, Side: op.Side, Char: op.Char,
			}))
		}
		Expect(dispatch(ctx, key, "dk", acts)).To(Succeed())
		Expect(svc.NewRetrieve().
			Where(arc.MatchKeys(key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		return stored
	}
	deploy := func(ctx SpecContext, a arc.Arc) task.Key {
		GinkgoHelper()
		var tsk *task.Task
		Expect(write(ctx, func(w arc.Writer) (err error) {
			tsk, err = w.UpdateTask(
				ctx, a.Base().Key, arc.TaskUpdate{Rack: new(testRack.Key)},
			)
			return err
		})).To(Succeed())
		return tsk.Key
	}
	createDirectory := func(
		ctx SpecContext,
		name string,
		parent directory.Key,
	) directory.Directory {
		GinkgoHelper()
		d := directory.Directory{Name: name}
		Expect(write(ctx, func(w arc.Writer) error {
			return w.CreateDirectory(ctx, &d, parent)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(write(ctx, func(w arc.Writer) error {
				return w.DeleteDirectory(ctx, d.Key)
			})).To(Succeed())
		})
		return d
	}
	move := func(ctx SpecContext, a arc.Arc, to directory.Key) {
		GinkgoHelper()
		Expect(write(ctx, func(w arc.Writer) error {
			return w.Move(ctx, root, to, a.ResourceID())
		})).To(Succeed())
	}
	configHash := func(ctx SpecContext, key task.Key) func() string {
		return func() string {
			var tsk task.Task
			if err := taskSvc.NewRetrieve().
				Where(task.MatchKeys(key)).
				Entry(&tsk).
				Exec(ctx, db); err != nil {
				return ""
			}
			return tsk.ConfigHash
		}
	}
	// settledHash waits out the resync that creating or deploying an Arc schedules
	// and returns the task's config hash once it holds still.
	settledHash := func(ctx SpecContext, key task.Key) string {
		GinkgoHelper()
		var hash string
		Eventually(func(g Gomega) {
			hash = configHash(ctx, key)()
			g.Consistently(configHash(ctx, key), 30*time.Millisecond, 5*time.Millisecond).
				Should(Equal(hash))
		}).Should(Succeed())
		return hash
	}
	// setup creates the module and a deployed importer and returns the importer's
	// task key.
	setup := func(ctx SpecContext) (arc.Arc, task.Key) {
		GinkgoHelper()
		m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
		a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
		return m, deploy(ctx, a)
	}
	BeforeEach(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			root = rootDirectory(ctx, tx).Key
			return nil
		})).To(Succeed())
	})
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		for name, dt := range map[string]telem.DataType{
			"pt_trig": telem.Uint8T,
			"pt_out":  telem.Int64T,
		} {
			ch := &channel.Channel{Name: name, Virtual: true, DataType: dt}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, ch)
			})).To(Succeed())
		}
	})

	Describe("Compiled hash", func() {
		It("Should hash equal programs equally", func(ctx SpecContext) {
			createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			first := MustSucceed(svc.CompileProgram(ctx, a.Base().Key))
			second := MustSucceed(svc.CompileProgram(ctx, a.Base().Key))
			Expect(MustSucceed(arc.HashProgram(*first.Program))).
				To(Equal(MustSucceed(arc.HashProgram(*second.Program))))
		})

		It("Should hash different programs differently", func(ctx SpecContext) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			first := MustSucceed(svc.CompileProgram(ctx, a.Base().Key))
			overwrite(ctx, m, moduleV2)
			second := MustSucceed(svc.CompileProgram(ctx, a.Base().Key))
			Expect(MustSucceed(arc.HashProgram(*first.Program))).
				ToNot(Equal(MustSucceed(arc.HashProgram(*second.Program))))
		})

		It("Should store the compiled hash in the task config", func(ctx SpecContext) {
			createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			var tsk task.Task
			Expect(taskSvc.NewRetrieve().
				Where(task.MatchKeys(key)).
				Entry(&tsk).
				Exec(ctx, db)).To(Succeed())
			compiled := MustSucceed(svc.CompileProgram(ctx, a.Base().Key))
			Expect(tsk.Config["hash"]).
				To(Equal(MustSucceed(arc.HashProgram(*compiled.Program))))
		})

		It("Should hash its content when the automation does not compile", func(
			ctx SpecContext,
		) {
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			var tsk task.Task
			Expect(taskSvc.NewRetrieve().
				Where(task.MatchKeys(key)).
				Entry(&tsk).
				Exec(ctx, db)).To(Succeed())
			records := MustSucceed(svc.RetrieveDrift(ctx, db, a.Base().Key))
			stored := records[a.Base().Key].Stored
			Expect(stored.Text).To(Equal(MustSucceed(arc.Hash(a.Base()))))
			Expect(tsk.Config["hash"]).To(Equal(MustSucceed(arc.HashContent(stored))))
		})
	})

	Describe("Module edits", func() {
		It("Should drift the importer when a module is overwritten", func(
			ctx SpecContext,
		) {
			m, key := setup(ctx)
			before := settledHash(ctx, key)
			overwrite(ctx, m, moduleV2)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})

		It("Should drift the importer when a module is edited", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", "")
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			client := crdt.New(2)
			var acts []arc.Action
			for _, op := range client.Insert(0, moduleV1) {
				acts = append(acts, arc.NewInsertCharAction(arc.InsertCharPayload{
					ID: op.ID, Origin: op.Origin, Side: op.Side, Char: op.Char,
				}))
			}
			Expect(dispatch(ctx, m.Base().Key, "dk", acts)).To(Succeed())
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})

		It("Should drift an importer two modules away", func(ctx SpecContext) {
			m := createArc(ctx, arc.TypeModule, "pt_inner", moduleV1)
			createArc(
				ctx,
				arc.TypeModule,
				"pt_m",
				"import pt_inner\n\nexport func f() {t u8} i64 { return pt_inner.f(t) }",
			)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			overwrite(ctx, m, moduleV2)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})

		It("Should leave an automation that does not import the module", func(
			ctx SpecContext,
		) {
			m, _ := setup(ctx)
			other := createArc(ctx, arc.TypeAutomation, "pt_other", "pt_trig -> pt_out")
			key := deploy(ctx, other)
			before := configHash(ctx, key)()
			overwrite(ctx, m, moduleV2)
			Consistently(
				configHash(ctx, key),
				50*time.Millisecond,
				5*time.Millisecond,
			).Should(Equal(before))
		})
	})

	Describe("Path changes", func() {
		It("Should drift the importer when the module is renamed", func(
			ctx SpecContext,
		) {
			m, key := setup(ctx)
			before := settledHash(ctx, key)
			Expect(dispatch(ctx, m.Base().Key, "", []arc.Action{
				arc.NewSetAction(
					arc.SetPayload{Path: []string{"name"}, Value: "pt_renamed"},
				),
			})).To(Succeed())
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})

		It("Should drift the importer when the module is moved", func(
			ctx SpecContext,
		) {
			d := createDirectory(ctx, "pt_lib", root)
			m, key := setup(ctx)
			before := settledHash(ctx, key)
			move(ctx, m, d.Key)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})

		It(
			"Should drift the importer when a dispatch sets the module's directory",
			func(ctx SpecContext) {
				d := createDirectory(ctx, "pt_lib", root)
				m, key := setup(ctx)
				before := settledHash(ctx, key)
				Expect(dispatch(ctx, m.Base().Key, "", []arc.Action{
					arc.NewSetDirectoryAction(
						arc.SetDirectoryPayload{Directory: d.Key},
					),
				})).To(Succeed())
				Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
			},
		)

		It("Should drift the importer when the module's directory is renamed", func(
			ctx SpecContext,
		) {
			d := createDirectory(ctx, "pt_lib", root)
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			move(ctx, m, d.Key)
			a := createArc(
				ctx,
				arc.TypeAutomation,
				"pt_a",
				"import pt_lib.pt_m\n\npt_trig -> pt_m.f() -> pt_out",
			)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			Expect(write(ctx, func(w arc.Writer) error {
				return w.RenameDirectory(ctx, d.Key, "pt_lib2")
			})).To(Succeed())
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})

		It("Should drift the importer when the module is deleted", func(
			ctx SpecContext,
		) {
			m := newModule(arc.Base{Name: "pt_m", Source: textSource(moduleV1)})
			Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &m) })).
				To(Succeed())
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			Expect(write(ctx, func(w arc.Writer) error {
				return w.Delete(ctx, m.Base().Key)
			})).To(Succeed())
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})

		It("Should drift the importer when its module is created", func(
			ctx SpecContext,
		) {
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})
	})

	Describe("Module status", func() {
		moduleStatus := func(ctx SpecContext, key arc.Key) func() arc.Status {
			return func() arc.Status {
				var stat arc.Status
				if err := statusSvc.NewRetrieve[struct{}]().
					Where(status.MatchOwners[struct{}](arc.ResourceID(key))).
					Entry(&stat).
					Exec(ctx, db); err != nil {
					return arc.Status{}
				}
				return stat
			}
		}
		// settledVariant waits for the module's status and returns its variant once it
		// holds still.
		settledVariant := func(ctx SpecContext, key arc.Key) status.Variant {
			GinkgoHelper()
			variant := func() status.Variant { return moduleStatus(ctx, key)().Variant }
			var settled status.Variant
			Eventually(func(g Gomega) {
				settled = variant()
				g.Expect(settled).ToNot(BeEmpty())
				g.Consistently(variant, 30*time.Millisecond, 5*time.Millisecond).
					Should(Equal(settled))
			}).Should(Succeed())
			return settled
		}

		It("Should store a success status for a module that compiles", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			Eventually(moduleStatus(ctx, m.Base().Key)).Should(SatisfyAll(
				HaveField("Variant", status.VariantSuccess),
				HaveField("Message", "Compiled"),
				HaveField("Name", "pt_m"),
			))
		})

		It("Should store an error status for a module that does not compile", func(
			ctx SpecContext,
		) {
			m := createArc(
				ctx,
				arc.TypeModule,
				"pt_m",
				"export func f() i64 { return x }",
			)
			Eventually(moduleStatus(ctx, m.Base().Key)).Should(SatisfyAll(
				HaveField("Variant", status.VariantError),
				HaveField("Message", "Failed to compile"),
				HaveField("Description", ContainSubstring("x")),
			))
		})

		It("Should update the status when an imported module breaks", func(
			ctx SpecContext,
		) {
			inner := createArc(ctx, arc.TypeModule, "pt_inner", moduleV1)
			m := createArc(
				ctx,
				arc.TypeModule,
				"pt_m",
				"import pt_inner\n\nexport func f() {t u8} i64 { return pt_inner.f(t) }",
			)
			Expect(settledVariant(ctx, m.Base().Key)).To(Equal(status.VariantSuccess))
			overwrite(ctx, inner, "export func g() i64 { return 1 }")
			Eventually(func() status.Variant {
				return moduleStatus(ctx, m.Base().Key)().Variant
			}).Should(Equal(status.VariantError))
		})

		It("Should update the status of an importer in a directory", func(
			ctx SpecContext,
		) {
			lib := createDirectory(ctx, "pt_lib", root)
			sub := createDirectory(ctx, "pt_sub", lib.Key)
			inner := createArc(ctx, arc.TypeModule, "pt_inner", moduleV1)
			move(ctx, inner, sub.Key)
			m := createArc(
				ctx,
				arc.TypeModule,
				"pt_m",
				"import pt_lib.pt_sub.pt_inner\n\n"+
					"export func f() {t u8} i64 { return pt_inner.f(t) }",
			)
			move(ctx, m, lib.Key)
			Expect(settledVariant(ctx, m.Base().Key)).To(Equal(status.VariantSuccess))
			overwrite(ctx, inner, "export func g() i64 { return 1 }")
			Eventually(func() status.Variant {
				return moduleStatus(ctx, m.Base().Key)().Variant
			}).Should(Equal(status.VariantError))
		})

		It("Should fail every module above a renamed export in a chain", func(
			ctx SpecContext,
		) {
			l1 := createDirectory(ctx, "pt_l1", root)
			l2 := createDirectory(ctx, "pt_l2", l1.Key)
			l3 := createDirectory(ctx, "pt_l3", l2.Key)
			c := createArc(
				ctx,
				arc.TypeModule,
				"pt_c",
				"export func value() i64 { return 40 }",
			)
			move(ctx, c, l3.Key)
			b := createArc(
				ctx,
				arc.TypeModule,
				"pt_b",
				"import pt_l1.pt_l2.pt_l3.pt_c\n\n"+
					"export doubled := pt_out * 2\n\n"+
					"export func value() i64 { return pt_c.value() + 1 }",
			)
			move(ctx, b, l2.Key)
			a := createArc(ctx, arc.TypeModule, "pt_a", "import pt_l1.pt_l2.pt_b\n\n"+
				"export func value() {t u8} i64 { return pt_b.value() + 1 }\n\n"+
				"export stage relay { pt_b.doubled + 1 -> pt_out }")
			move(ctx, a, l1.Key)
			auto := createArc(
				ctx,
				arc.TypeAutomation,
				"pt_auto",
				"import pt_l1.pt_a\n\npt_trig -> pt_a.value() -> pt_out\n"+
					"pt_trig == 1 => pt_a.relay",
			)
			key := deploy(ctx, auto)
			before := configHash(ctx, key)()
			for _, m := range []arc.Arc{a, b, c} {
				Expect(
					settledVariant(ctx, m.Base().Key),
				).To(Equal(status.VariantSuccess))
			}
			overwrite(ctx, c, "export func renamed() i64 { return 40 }")
			for _, m := range []arc.Arc{b, a} {
				Eventually(func() status.Variant {
					return moduleStatus(ctx, m.Base().Key)().Variant
				}).Should(Equal(status.VariantError))
			}
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})

		It("Should delete the status with the module", func(ctx SpecContext) {
			m := newModule(arc.Base{Name: "pt_m", Source: textSource(moduleV1)})
			Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &m) })).
				To(Succeed())
			Eventually(
				moduleStatus(ctx, m.Base().Key),
			).ShouldNot(HaveField("Key", BeEmpty()))
			Expect(write(ctx, func(w arc.Writer) error {
				return w.Delete(ctx, m.Base().Key)
			})).To(Succeed())
			Expect(moduleStatus(ctx, m.Base().Key)().Key).To(BeEmpty())
		})
	})

	Describe("Deploy record", func() {
		const broken = "import pt_m\n\npt_trig -> pt_m.g() -> pt_out"
		record := func(ctx SpecContext, a arc.Arc) drift.Record {
			GinkgoHelper()
			records := MustSucceed(svc.RetrieveDrift(ctx, db, a.Base().Key))
			Expect(records).To(HaveKey(a.Base().Key))
			return records[a.Base().Key]
		}
		hashOf := func(a arc.Arc) string {
			GinkgoHelper()
			return MustSucceed(arc.Hash(a.Base()))
		}
		// run reports the task as running the config it holds, as the Driver does.
		run := func(ctx SpecContext, key task.Key) {
			GinkgoHelper()
			stat := &task.Status{
				Owner:   new(task.ResourceID(key)),
				Name:    "pt_a",
				Variant: status.VariantSuccess,
				Time:    telem.Now(),
				Details: task.StatusDetails{
					Running:    true,
					ConfigHash: configHash(ctx, key)(),
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return statusSvc.NewWriter(tx).Set(ctx, stat)
			})).To(Succeed())
		}

		It("Should record the text hash and each module's hash at deploy", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			deploy(ctx, a)
			rec := record(ctx, a)
			Expect(rec.Stored.Text).To(Equal(hashOf(a)))
			Expect(rec.Stored.Modules).To(Equal(map[string]string{"pt_m": hashOf(m)}))
			Expect(rec.Deployed).To(Equal(drift.Hashes{}))
		})

		It("Should record a module imported through another module", func(
			ctx SpecContext,
		) {
			const outer = "import pt_inner\n\n" +
				"export func f() {t u8} i64 { return pt_inner.f(t) }"
			inner := createArc(ctx, arc.TypeModule, "pt_inner", moduleV1)
			m := createArc(ctx, arc.TypeModule, "pt_m", outer)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			deploy(ctx, a)
			Expect(record(ctx, a).Stored.Modules).To(Equal(map[string]string{
				"pt_inner": hashOf(inner),
				"pt_m":     hashOf(m),
			}))
		})

		It("Should keep the deployed hashes empty until the Driver runs", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			edited := overwrite(ctx, m, moduleV2)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
			rec := record(ctx, a)
			Expect(rec.Stored.Modules["pt_m"]).To(Equal(hashOf(edited)))
			Expect(rec.Deployed).To(Equal(drift.Hashes{}))
		})

		It("Should promote the stored hashes once the Driver runs them", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			run(ctx, key)
			edited := overwrite(ctx, m, moduleV2)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
			rec := record(ctx, a)
			Expect(rec.Deployed.Text).To(Equal(hashOf(a)))
			Expect(rec.Deployed.Modules["pt_m"]).To(Equal(hashOf(m)))
			Expect(rec.Stored.Modules["pt_m"]).To(Equal(hashOf(edited)))
		})

		It("Should not promote while the Driver runs an older config", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			first := settledHash(ctx, key)
			run(ctx, key)
			overwrite(ctx, m, moduleV2)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(first))
			second := settledHash(ctx, key)
			restored := overwrite(ctx, m, moduleV1)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(second))
			rec := record(ctx, a)
			Expect(rec.Deployed.Modules["pt_m"]).To(Equal(hashOf(m)))
			Expect(rec.Stored.Modules["pt_m"]).To(Equal(hashOf(restored)))
		})

		It("Should name nothing once the Driver runs the edit", func(ctx SpecContext) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			run(ctx, key)
			overwrite(ctx, m, moduleV2)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
			Expect(arc.Changes(record(ctx, a))).ToNot(BeEmpty())
			run(ctx, key)
			Expect(arc.Changes(record(ctx, a))).To(BeEmpty())
		})

		It("Should keep the module hashes when the automation stops compiling", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			edited := overwrite(ctx, a, broken)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
			rec := record(ctx, a)
			Expect(rec.Stored.Text).To(Equal(hashOf(edited)))
			Expect(rec.Stored.Modules).To(Equal(map[string]string{"pt_m": hashOf(m)}))
		})

		It("Should rehash a module whose edit breaks the automation", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			edited := overwrite(ctx, m, "export func g() {t u8} i64 { return 1 }")
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
			rec := record(ctx, a)
			Expect(rec.Stored.Text).To(Equal(hashOf(a)))
			Expect(
				rec.Stored.Modules,
			).To(Equal(map[string]string{"pt_m": hashOf(edited)}))
		})

		It("Should rehash a module edited while the automation does not compile", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			a = overwrite(ctx, a, broken)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
			brokenHash := settledHash(ctx, key)
			edited := overwrite(ctx, m, moduleV2)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(brokenHash))
			Expect(record(ctx, a).Stored.Modules["pt_m"]).To(Equal(hashOf(edited)))
		})

		It("Should leave the record when a module edit keeps the program", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			overwrite(ctx, m, moduleV1+"\n// unchanged")
			Consistently(func() map[string]string {
				return record(ctx, a).Stored.Modules
			}, 50*time.Millisecond, 5*time.Millisecond).Should(
				Equal(map[string]string{"pt_m": hashOf(m)}),
			)
			Expect(configHash(ctx, key)()).To(Equal(before))
		})

		It("Should drop a module that left its path from a broken automation", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := settledHash(ctx, key)
			Expect(dispatch(ctx, m.Base().Key, "", []arc.Action{
				arc.NewSetAction(
					arc.SetPayload{Path: []string{"name"}, Value: "pt_gone"},
				),
			})).To(Succeed())
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
			Expect(record(ctx, a).Stored.Modules).To(BeEmpty())
		})

		It("Should name the automation first, then each module by path", func() {
			Expect(arc.Changes(drift.Record{
				Stored: drift.Hashes{
					Text: "t2",
					Modules: map[string]string{
						"pt_b": "b2",
						"pt_c": "c1",
						"pt_d": "d1",
					},
				},
				Deployed: drift.Hashes{
					Text: "t1",
					Modules: map[string]string{
						"pt_a": "a1",
						"pt_b": "b1",
						"pt_c": "c1",
					},
				},
			})).To(Equal([]arc.Change{
				{Type: arc.TypeAutomation, Kind: arc.ChangeKindChanged},
				{Type: arc.TypeModule, Path: "pt_a", Kind: arc.ChangeKindRemoved},
				{Type: arc.TypeModule, Path: "pt_b", Kind: arc.ChangeKindChanged},
				{Type: arc.TypeModule, Path: "pt_d", Kind: arc.ChangeKindAdded},
			}))
		})

		It("Should name nothing before the task is deployed", func() {
			hashes := drift.Hashes{Text: "t1", Modules: map[string]string{"pt_m": "m1"}}
			Expect(arc.Changes(drift.Record{Stored: hashes})).To(BeEmpty())
		})

		It("Should name nothing when the task runs what is stored", func() {
			hashes := drift.Hashes{Text: "t1", Modules: map[string]string{"pt_m": "m1"}}
			Expect(
				arc.Changes(drift.Record{Stored: hashes, Deployed: hashes}),
			).To(BeEmpty())
		})

		It("Should delete the record when the automation is unbound", func(
			ctx SpecContext,
		) {
			createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			deploy(ctx, a)
			record(ctx, a)
			Expect(write(ctx, func(w arc.Writer) error {
				tsk, err := w.UpdateTask(
					ctx, a.Base().Key, arc.TaskUpdate{Rack: new(rack.Key)},
				)
				Expect(tsk).To(BeNil())
				return err
			})).To(Succeed())
			Expect(MustSucceed(svc.RetrieveDrift(ctx, db, a.Base().Key))).To(BeEmpty())
		})

		It("Should delete the record with the automation", func(ctx SpecContext) {
			a := newArc(arc.TypeAutomation, arc.Base{
				Name: "pt_a", Source: textSource("pt_trig -> pt_out"),
			})
			Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &a) })).
				To(Succeed())
			deploy(ctx, a)
			record(ctx, a)
			Expect(write(ctx, func(w arc.Writer) error {
				return w.Delete(ctx, a.Base().Key)
			})).To(Succeed())
			Expect(MustSucceed(svc.RetrieveDrift(ctx, db, a.Base().Key))).To(BeEmpty())
		})
	})

	Describe("Language server", func() {
		It("Should republish an importer's diagnostics when a module changes", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			server := MustSucceed(svc.NewLSP(lspConfig))
			DeferCleanup(func() { Expect(server.Shutdown(ctx)).To(Succeed()) })
			client := &MockClient{}
			server.SetClient(client)
			OpenArcDocument(server, ctx, uri.URI("file:///pt_a.arc"), importer)
			Expect(client.Diagnostics()).To(BeEmpty())
			overwrite(ctx, m, "export func g() i64 { return 1 }")
			Eventually(client.Diagnostics).ShouldNot(BeEmpty())
		})

		It("Should republish an importer's diagnostics on a directory rename", func(
			ctx SpecContext,
		) {
			d := createDirectory(ctx, "pt_lib", root)
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			move(ctx, m, d.Key)
			server := MustSucceed(svc.NewLSP(lspConfig))
			DeferCleanup(func() { Expect(server.Shutdown(ctx)).To(Succeed()) })
			client := &MockClient{}
			server.SetClient(client)
			OpenArcDocument(
				server,
				ctx,
				uri.URI("file:///pt_a.arc"),
				"import pt_lib.pt_m\n\npt_trig -> pt_m.f() -> pt_out",
			)
			Expect(client.Diagnostics()).To(BeEmpty())
			Expect(write(ctx, func(w arc.Writer) error {
				return w.RenameDirectory(ctx, d.Key, "pt_core")
			})).To(Succeed())
			Eventually(client.Diagnostics).ShouldNot(BeEmpty())
		})
	})
})
