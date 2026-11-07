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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
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
	overwrite := func(ctx SpecContext, a arc.Arc, raw string) {
		GinkgoHelper()
		b := a.Base()
		b.Source = textSource(raw)
		a.SetBase(b)
		Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &a) })).
			To(Succeed())
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
	createDirectory := func(ctx SpecContext, name string) directory.Directory {
		GinkgoHelper()
		d := directory.Directory{Name: name}
		Expect(write(ctx, func(w arc.Writer) error {
			return w.CreateDirectory(ctx, &d, root)
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

		It("Should hash the source when the automation does not compile", func(
			ctx SpecContext,
		) {
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			var tsk task.Task
			Expect(taskSvc.NewRetrieve().
				Where(task.MatchKeys(key)).
				Entry(&tsk).
				Exec(ctx, db)).To(Succeed())
			Expect(tsk.Config["hash"]).To(Equal(MustSucceed(arc.Hash(a.Base()))))
		})
	})

	Describe("Module edits", func() {
		It("Should drift the importer when a module is overwritten", func(
			ctx SpecContext,
		) {
			m, key := setup(ctx)
			before := configHash(ctx, key)()
			overwrite(ctx, m, moduleV2)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})

		It("Should drift the importer when a module is edited", func(
			ctx SpecContext,
		) {
			m := createArc(ctx, arc.TypeModule, "pt_m", "")
			a := createArc(ctx, arc.TypeAutomation, "pt_a", importer)
			key := deploy(ctx, a)
			before := configHash(ctx, key)()
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
			before := configHash(ctx, key)()
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
			before := configHash(ctx, key)()
			Expect(dispatch(ctx, m.Base().Key, "", []arc.Action{
				arc.NewRenameAction(arc.RenamePayload{Name: "pt_renamed"}),
			})).To(Succeed())
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})

		It("Should drift the importer when the module is moved", func(
			ctx SpecContext,
		) {
			d := createDirectory(ctx, "pt_lib")
			m, key := setup(ctx)
			before := configHash(ctx, key)()
			move(ctx, m, d.Key)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
		})

		It("Should drift the importer when the module's directory is renamed", func(
			ctx SpecContext,
		) {
			d := createDirectory(ctx, "pt_lib")
			m := createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			move(ctx, m, d.Key)
			a := createArc(
				ctx,
				arc.TypeAutomation,
				"pt_a",
				"import pt_lib.pt_m\n\npt_trig -> pt_m.f() -> pt_out",
			)
			key := deploy(ctx, a)
			before := configHash(ctx, key)()
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
			before := configHash(ctx, key)()
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
			before := configHash(ctx, key)()
			createArc(ctx, arc.TypeModule, "pt_m", moduleV1)
			Eventually(configHash(ctx, key)).ShouldNot(Equal(before))
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
	})
})
