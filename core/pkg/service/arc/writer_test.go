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
	"encoding/json"
	"io"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create an Arc with generated key", func(ctx SpecContext) {
			a := arc.Arc{Name: "test-arc", Mode: arc.ModeText}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should return a validation error when the name is empty", func(ctx SpecContext) {
			a := arc.Arc{Mode: arc.ModeText}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).
				To(MatchError(ContainSubstring("name: required")))
		})

		It("Should create an Arc with explicit key", func(ctx SpecContext) {
			key := uuid.New()
			a := arc.Arc{Key: key, Name: "test-arc-with-key", Mode: arc.ModeText}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Key).To(Equal(key))
		})

		It("Should handle multiple arc creations", func(ctx SpecContext) {
			a1 := arc.Arc{Name: "arc-1", Mode: arc.ModeText}
			a2 := arc.Arc{Name: "arc-2", Mode: arc.ModeText}
			Expect(svc.NewWriter(tx).Create(ctx, &a1)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &a2)).To(Succeed())
			Expect(a1.Key).ToNot(Equal(uuid.Nil))
			Expect(a2.Key).ToNot(Equal(uuid.Nil))
			Expect(a1.Key).ToNot(Equal(a2.Key))
		})
	})

	Describe("CreateMany", func() {
		It("Should create multiple Arcs", func(ctx SpecContext) {
			arcs := []arc.Arc{
				{Name: "arc-many-1", Mode: arc.ModeText},
				{Name: "arc-many-2", Mode: arc.ModeText},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &arcs)).To(Succeed())

			var retrieved []arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(
				arcs[0].Key,
				arcs[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})

	Describe("Update", func() {
		It("Should update an existing Arc", func(ctx SpecContext) {
			key := uuid.New()
			a := arc.Arc{
				Key:   key,
				Name:  "existing-arc",
				Mode:  arc.ModeText,
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			a.Name = "updated-arc"
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			var retrieved arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(key)).Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Name).To(Equal("updated-arc"))
		})
	})

	Describe("Delete", func() {
		It("Should delete an Arc", func(ctx SpecContext) {
			a := arc.Arc{
				Name:  "arc-to-delete",
				Mode:  arc.ModeText,
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple Arcs", func(ctx SpecContext) {
			a1 := arc.Arc{Name: "arc-to-delete-1", Mode: arc.ModeText}
			a2 := arc.Arc{Name: "arc-to-delete-2", Mode: arc.ModeText}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &a1)).To(Succeed())
			Expect(w.Create(ctx, &a2)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a1.Key, a2.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(a1.Key, a2.Key)).
				Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
		})

		It("Should handle delete of non-existent arc gracefully", func(ctx SpecContext) {
			nonExistentKey := uuid.New()
			Expect(svc.NewWriter(tx).Delete(ctx, nonExistentKey)).To(Succeed())
		})

		It("Should delete child tasks when deleting an arc", func(ctx SpecContext) {
			a := arc.Arc{Name: "arc-with-task", Mode: arc.ModeText}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			t := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "child-task",
				Type: "arc",
			}
			Expect(taskSvc.NewWriter(tx).Create(ctx, t)).To(Succeed())

			Expect(otg.NewWriter(tx).DefineRelationship(
				ctx,
				arc.OntologyID(a.Key),
				ontology.RelationshipTypeParentOf,
				task.OntologyID(t.Key),
			)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())

			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(t.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should handle arc deletion when arc has no child tasks", func(ctx SpecContext) {
			a := arc.Arc{Name: "arc-without-tasks", Mode: arc.ModeText}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple child tasks when deleting an arc", func(ctx SpecContext) {
			a := arc.Arc{Name: "arc-with-multiple-tasks", Mode: arc.ModeText}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			t1 := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "child-task-1",
				Type: "arc",
			}
			t2 := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "child-task-2",
				Type: "arc",
			}
			Expect(taskSvc.NewWriter(tx).Create(ctx, t1)).To(Succeed())
			Expect(taskSvc.NewWriter(tx).Create(ctx, t2)).To(Succeed())

			otgWriter := otg.NewWriter(tx)
			Expect(otgWriter.DefineRelationship(
				ctx,
				arc.OntologyID(a.Key),
				ontology.RelationshipTypeParentOf,
				task.OntologyID(t1.Key),
			)).To(Succeed())
			Expect(otgWriter.DefineRelationship(
				ctx,
				arc.OntologyID(a.Key),
				ontology.RelationshipTypeParentOf,
				task.OntologyID(t2.Key),
			)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())

			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(t1.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(t2.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Dispatch", func() {
		var (
			setChannel channel.Channel
			requests   confluence.Inlet[framer.StreamerRequest]
			responses  confluence.Outlet[framer.StreamerResponse]
			shutdown   io.Closer
		)
		BeforeEach(func(ctx SpecContext) {
			Expect(dist.Channel.NewRetrieve().
				Where(channel.MatchNames("sy_arc_set")).
				Entry(&setChannel).
				Exec(ctx, nil)).To(Succeed())
			streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
				Keys: channel.Keys{setChannel.Key()},
			}))
			requests, responses = confluence.Attach(streamer, 2)
			sCtx, cancel := signal.Isolated()
			shutdown = signal.NewHardShutdown(sCtx, cancel)
			DeferCleanup(func() {
				requests.Close()
				confluence.Drain(responses)
				Expect(shutdown.Close()).To(Succeed())
			})
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(10 * time.Millisecond)
		})

		insert := func(replica uint32, counter uint64, char rune) arc.Action {
			return arc.NewInsertCharAction(arc.InsertCharPayload{Op: crdt.Insert{
				ID:   crdt.ID{Replica: replica, Counter: counter},
				Side: spatial.XLocationRight,
				Char: char,
			}})
		}

		It("Should broadcast dispatched actions on the arc set channel", func(ctx SpecContext) {
			key := uuid.New()
			seq := []arc.Action{
				insert(1, 1, 'h'),
				arc.NewDeleteCharAction(arc.DeleteCharPayload{Op: crdt.Delete{
					ID: crdt.ID{Replica: 1, Counter: 1},
				}}),
			}
			Expect(svc.NewWriter(nil).Dispatch(ctx, key, "dk-1", seq)).To(Succeed())
			var res framer.StreamerResponse
			Eventually(responses.Outlet(), time.Second*5).Should(Receive(&res))
			var decoded []actions.Scoped[arc.Key, arc.Action]
			for sample := range res.Frame.SeriesAt(0).Samples() {
				var sa actions.Scoped[arc.Key, arc.Action]
				Expect(json.Unmarshal(sample, &sa)).To(Succeed())
				decoded = append(decoded, sa)
			}
			Expect(decoded).To(HaveLen(1))
			Expect(decoded[0].Key).To(Equal(key))
			Expect(decoded[0].DispatchKey).To(Equal("dk-1"))
			Expect(decoded[0].Actions).To(Equal(seq))
		})

		It("Should scope each broadcast to the dispatched arc key", func(ctx SpecContext) {
			keyA, keyB := uuid.New(), uuid.New()
			Expect(svc.NewWriter(nil).Dispatch(ctx, keyA, "", []arc.Action{insert(1, 1, 'a')})).To(Succeed())
			Expect(svc.NewWriter(nil).Dispatch(ctx, keyB, "", []arc.Action{insert(2, 1, 'b')})).To(Succeed())
			var keys []arc.Key
			Eventually(func(g Gomega) []arc.Key {
				select {
				case res := <-responses.Outlet():
					for sample := range res.Frame.SeriesAt(0).Samples() {
						var sa actions.Scoped[arc.Key, arc.Action]
						g.Expect(json.Unmarshal(sample, &sa)).To(Succeed())
						keys = append(keys, sa.Key)
					}
				default:
				}
				return keys
			}, time.Second*5).Should(ConsistOf(keyA, keyB))
		})
	})
})
