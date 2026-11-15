// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[secret.Key, secret.Action]
	var (
		e *env
		s secret.Secret
	)
	write := func(ctx context.Context, fn func(secret.Writer) error) error {
		return e.write(ctx, func(w secret.Writer) error {
			return fn(w.WithDispatchKey("dk"))
		})
	}
	set := func(path string, value any) secret.Action {
		return secret.NewSetAction(
			secret.SetPayload{Path: []string{path}, Value: value},
		)
	}
	apply := func(ctx context.Context, acts ...secret.Action) error {
		return write(ctx, func(w secret.Writer) error {
			return w.Apply(ctx, s.Base().Key, acts...)
		})
	}
	record := func() *Recorder[secret.Key, secret.Action] {
		rec := &Recorder[secret.Key, secret.Action]{}
		DeferCleanup(e.svc.Actions().OnChange(rec.Record))
		return rec
	}
	BeforeEach(func(ctx SpecContext) {
		e = newEnv(ctx)
		s = e.createStored(ctx, "Before", "hunter2")
	})

	It("Should store the edited name and keep the value", func(ctx SpecContext) {
		Expect(apply(ctx, set("name", "After"))).To(Succeed())
		got := e.retrieve(ctx, s.Base().Key)
		Expect(got.Base().Name).To(Equal("After"))
		Expect(got.Variant.(secret.StoredSecret).Value).
			To(Equal(s.Variant.(secret.StoredSecret).Value))
	})

	DescribeTable(
		"Should refuse an edit",
		func(
			ctx SpecContext,
			act secret.Action,
			matcher string,
		) {
			err := apply(ctx, act)
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring(matcher)))
			Expect(e.retrieve(ctx, s.Base().Key)).To(Equal(s))
		},
		Entry("of the value", set("value", "x"), `unknown object member name "value"`),
		Entry(
			"of the source",
			set("source", "file"),
			`unknown object member name "source"`,
		),
		Entry(
			"that creates the secret",
			secret.NewCreateAction(secret.CreatePayload{}),
			"a secret create",
		),
		Entry(
			"that removes the name",
			secret.NewRemoveAction(secret.RemovePayload{Key: "name"}),
			"[name]: the document requires a value",
		),
		Entry(
			"that inserts into the name",
			secret.NewInsertAction(secret.InsertPayload{
				Path:  []string{"name"},
				Value: map[string]any{"key": "a"},
			}),
			"[name]: the value is not a list",
		),
		Entry(
			"that moves within the name",
			secret.NewMoveAction(secret.MovePayload{Path: []string{"name"}, Key: "a"}),
			"[name]: the value is not a list",
		),
	)

	It("Should refuse an empty name", func(ctx SpecContext) {
		Expect(
			apply(ctx, set("name", "")),
		).To(MatchError(ContainSubstring("name: required")))
		Expect(e.retrieve(ctx, s.Base().Key)).To(Equal(s))
	})

	It("Should broadcast a create without the value", func(ctx SpecContext) {
		rec := record()
		created := secret.Secret{Variant: secret.StoredSecret{
			Base: secret.Base{Name: "Created"},
		}}
		Expect(write(ctx, func(w secret.Writer) error {
			return w.Set(ctx, &created, []byte("hunter2"))
		})).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Actions).To(HaveLen(1))
		payload := frames[0].Actions[0].Create.Secret
		Expect(payload.Base()).To(Equal(created.Base()))
		Expect(payload.Variant.(secret.StoredSecret).Value).To(BeNil())
	})

	It(
		"Should broadcast an applied batch with its dispatch key",
		func(ctx SpecContext) {
			rec := record()
			Expect(apply(ctx, set("name", "After"))).To(Succeed())
			frames := rec.Snapshot()
			Expect(frames).To(Equal([]frame{{
				Key:         s.Base().Key,
				DispatchKey: "dk",
				Seq:         frames[0].Seq,
				Actions:     []secret.Action{set("name", "After")},
			}}))
		},
	)

	It("Should broadcast the set that a rename applies", func(ctx SpecContext) {
		rec := record()
		Expect(write(ctx, func(w secret.Writer) error {
			return w.Rename(ctx, s.Base().Key, "After")
		})).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Actions).To(Equal([]secret.Action{set("name", "After")}))
	})
})
