// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Apply", func() {
	type frame = actions.Scoped[icd.Key, icd.Action]
	var (
		editDB  *gorp.DB
		editSvc *icd.Service
		bus     icd.ICD
	)
	// field returns a binary field in the form a JSON round trip leaves it.
	field := func(name string, startBit uint16) icd.BinaryField {
		f := binaryField(name, startBit, 8)
		f.Key = uuid.New()
		f.MultiplexValues = []int32{}
		return f
	}
	BeforeEach(func(ctx SpecContext) {
		editDB = DeferClose(gorp.Wrap(memkv.New()))
		editSvc = MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
			DB:   editDB,
			ImEx: imex.NewService(),
		}))
		bus = icd.ICD{Name: "Bus", Messages: []icd.Message{
			canMessage("Engine", 0x100, field("rpm", 0), field("temp", 8)),
			canMessage("Brakes", 0x101, field("pressure", 0)),
		}}
		Expect(editDB.WithTx(ctx, func(tx gorp.Tx) error {
			return editSvc.NewWriter(tx).Create(ctx, &bus)
		})).To(Succeed())
	})
	record := func() *Recorder[icd.Key, icd.Action] {
		rec := &Recorder[icd.Key, icd.Action]{}
		DeferCleanup(editSvc.OnAction(rec.Record))
		return rec
	}
	retrieve := func(ctx context.Context) icd.ICD {
		GinkgoHelper()
		var res icd.ICD
		Expect(editSvc.NewRetrieve().
			Where(icd.MatchKeys(bus.Key)).
			Entry(&res).
			Exec(ctx, editDB)).To(Succeed())
		return res
	}
	apply := func(
		ctx context.Context,
		key icd.Key,
		dispatchKey string,
		acts ...icd.Action,
	) error {
		return editDB.WithTx(ctx, func(tx gorp.Tx) error {
			return editSvc.NewWriter(tx).
				WithDispatchKey(dispatchKey).
				Apply(ctx, key, acts...)
		})
	}
	dispatch := func(ctx context.Context, acts ...icd.Action) error {
		return apply(ctx, bus.Key, "dk", acts...)
	}
	set := func(path []string, value any, guards ...patch.Guard) icd.Action {
		return icd.NewSetAction(icd.SetPayload{
			Path:   path,
			Value:  value,
			Guards: guards,
		})
	}
	messagePath := func(i int, segments ...string) []string {
		return append([]string{"messages", bus.Messages[i].Key.String()}, segments...)
	}
	fieldPath := func(i, j int, segments ...string) []string {
		key := binaryFieldOf(bus.Messages[i], j).Key.String()
		return append(messagePath(i, "payload", "fields", key), segments...)
	}
	formatGuard := func(i int, format icd.PayloadType) patch.Guard {
		return patch.Guard{
			Path:  messagePath(i, "payload", "format"),
			Value: string(format),
		}
	}
	// withScale returns bus with the scale of field j of message i set.
	withScale := func(i, j int, scale float64) icd.ICD {
		next := bus
		next.Messages = append([]icd.Message(nil), bus.Messages...)
		next.Messages[i] = updateBinary(next.Messages[i], func(p *icd.BinaryPayload) {
			p.Fields = append([]icd.BinaryField(nil), p.Fields...)
			p.Fields[j].Scale = scale
		})
		return next
	}

	It("Should rename the ICD", func(ctx SpecContext) {
		Expect(dispatch(ctx, set([]string{"name"}, "Vehicle"))).To(Succeed())
		want := bus
		want.Name = "Vehicle"
		Expect(retrieve(ctx)).To(Equal(want))
	})

	It("Should insert a message after another", func(ctx SpecContext) {
		doors := canMessage("Doors", 0x102, field("open", 0))
		doors.Key = uuid.New()
		after := bus.Messages[0].Key.String()
		Expect(dispatch(ctx, icd.NewInsertAction(icd.InsertPayload{
			Path:  []string{"messages"},
			Value: doors,
			After: &after,
		}))).To(Succeed())
		Expect(retrieve(ctx).Messages).To(Equal([]icd.Message{
			bus.Messages[0],
			doors,
			bus.Messages[1],
		}))
	})

	It("Should set a value of a field in a binary payload", func(ctx SpecContext) {
		Expect(dispatch(ctx, set(
			fieldPath(0, 1, "scale"),
			0.5,
			formatGuard(0, icd.BinaryPayloadType),
		))).To(Succeed())
		Expect(retrieve(ctx)).To(Equal(withScale(0, 1, 0.5)))
	})

	It("Should remove a message", func(ctx SpecContext) {
		Expect(dispatch(ctx, icd.NewRemoveAction(icd.RemovePayload{
			Path: []string{"messages"},
			Key:  bus.Messages[0].Key.String(),
		}))).To(Succeed())
		Expect(retrieve(ctx).Messages).To(Equal(bus.Messages[1:]))
	})

	It("Should move a message to the front", func(ctx SpecContext) {
		Expect(dispatch(ctx, icd.NewMoveAction(icd.MovePayload{
			Path: []string{"messages"},
			Key:  bus.Messages[1].Key.String(),
		}))).To(Succeed())
		Expect(retrieve(ctx).Messages).To(Equal([]icd.Message{
			bus.Messages[1],
			bus.Messages[0],
		}))
	})

	It("Should do nothing when the payload format guard does not match", func(
		ctx SpecContext,
	) {
		Expect(dispatch(ctx, set(
			fieldPath(0, 1, "scale"),
			0.5,
			formatGuard(0, icd.TextPayloadType),
		))).To(Succeed())
		Expect(retrieve(ctx)).To(Equal(bus))
	})

	It("Should store an ICD with problems that Check finds", func(ctx SpecContext) {
		Expect(dispatch(ctx, set(messagePath(1, "name"), "Engine"))).To(Succeed())
		res := retrieve(ctx)
		Expect(res.Messages[1].Name).To(Equal("Engine"))
		Expect(icd.Check(res)).To(ConsistOf(validate.Problem{
			Path:    "messages.1.name",
			Message: `duplicate message name "Engine"`,
		}))
	})

	DescribeTable("Should reject a result that breaks a rule of the schema",
		func(ctx SpecContext, act func() icd.Action, message string) {
			rec := record()
			Expect(dispatch(ctx, set([]string{"name"}, "Vehicle"), act())).
				To(MatchError(message))
			Expect(retrieve(ctx)).To(Equal(bus))
			Expect(rec.Snapshot()).To(BeEmpty())
		},
		Entry("an empty name", func() icd.Action {
			return set([]string{"name"}, "")
		}, "name: required"),
		Entry("a field with no bits", func() icd.Action {
			return set(fieldPath(0, 0, "bit_length"), 0)
		}, "messages.0.payload.fields.0.bit_length: "+
			"must be greater than or equal to 1"),
	)

	It("Should reject a value the ICD cannot hold", func(ctx SpecContext) {
		Expect(dispatch(ctx, set(fieldPath(0, 0, "bit_length"), "wide"))).
			To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("decoding document")),
			))
		Expect(retrieve(ctx)).To(Equal(bus))
	})

	It("Should return not found for a missing ICD", func(ctx SpecContext) {
		Expect(apply(ctx, uuid.New(), "dk", set([]string{"name"}, "Vehicle"))).
			To(MatchError(query.ErrNotFound))
	})

	It("Should broadcast the actions with the dispatch key", func(ctx SpecContext) {
		rec := record()
		acts := []icd.Action{set([]string{"name"}, "Vehicle")}
		Expect(apply(ctx, bus.Key, "dk-1", acts...)).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		f := frames[0]
		Expect(f.Key).To(Equal(bus.Key))
		Expect(f.DispatchKey).To(Equal("dk-1"))
		Expect(f.Seq).To(BeNumerically(">", 0))
		Expect(f.Actions).To(Equal(acts))
	})

	Describe("Writer", func() {
		It("Should broadcast a create with the dispatch key after the commit", func(
			ctx SpecContext,
		) {
			rec := record()
			ic := icd.ICD{Name: "Created"}
			editTx := DeferClose(editDB.OpenTx())
			w := editSvc.NewWriter(editTx).WithDispatchKey("dk-create")
			Expect(w.Create(ctx, &ic)).To(Succeed())
			Expect(rec.Snapshot()).To(BeEmpty())
			Expect(editTx.Commit(ctx)).To(Succeed())
			frames := rec.Snapshot()
			Expect(frames).To(ConsistOf(frame{
				Key:         ic.Key,
				DispatchKey: "dk-create",
				Seq:         frames[0].Seq,
				Actions: []icd.Action{
					icd.NewCreateAction(icd.CreatePayload{ICD: ic}),
				},
			}))
		})

		It("Should broadcast an import as a create with the dispatch key", func(
			ctx SpecContext,
		) {
			rec := record()
			var res icd.ICD
			Expect(editDB.WithTx(ctx, func(tx gorp.Tx) (err error) {
				res, err = editSvc.NewWriter(tx).
					WithDispatchKey("dk-import").
					Import(ctx, bus.Key, icd.FormatDBC, []byte(importDBC))
				return err
			})).To(Succeed())
			frames := rec.Snapshot()
			Expect(frames).To(ConsistOf(frame{
				Key:         bus.Key,
				DispatchKey: "dk-import",
				Seq:         frames[0].Seq,
				Actions: []icd.Action{
					icd.NewCreateAction(icd.CreatePayload{ICD: res}),
				},
			}))
		})

		It("Should not broadcast a create that never commits", func(ctx SpecContext) {
			rec := record()
			editTx := editDB.OpenTx()
			ic := icd.ICD{Name: "Aborted"}
			Expect(editSvc.NewWriter(editTx).Create(ctx, &ic)).To(Succeed())
			Expect(editTx.Close()).To(Succeed())
			Expect(rec.Snapshot()).To(BeEmpty())
		})
	})
})
