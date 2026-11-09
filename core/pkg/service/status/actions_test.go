// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Actions", func() {
	var (
		db     *gorp.DB
		labels *label.Service
		svc    *status.Service
		rec    *Recorder[status.Key, status.Action]
		stored status.Status[any]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		labels = MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		svc = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		rec = &Recorder[status.Key, status.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		stored = status.Status[any]{
			Name:    "Pump",
			Variant: status.VariantSuccess,
			Message: "Pump running",
			Time:    telem.SecondTS,
			Details: map[string]any{"rpm": 1200.0},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Set(ctx, &stored)
		})).To(Succeed())
	})
	// sets returns the statuses that the recorded set actions carry, decoded from the
	// JSON that the action channel sends.
	sets := func() []status.Status[any] {
		GinkgoHelper()
		var out []status.Status[any]
		for _, f := range rec.Snapshot() {
			for _, a := range f.Actions {
				set, ok := a.Variant.(status.SetPayload)
				Expect(ok).To(BeTrue())
				var s status.Status[any]
				Expect(json.Unmarshal(MustSucceed(json.Marshal(set.Status)), &s)).
					To(Succeed())
				out = append(out, s)
			}
		}
		return out
	}

	It("Should attach a set that holds the stored status", func() {
		Expect(sets()).To(Equal([]status.Status[any]{stored}))
	})

	It("Should attach a set of the status a label delete leaves", func(
		ctx SpecContext,
	) {
		l := label.Label{Name: "Critical"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := labels.NewWriter(tx).Create(ctx, &l); err != nil {
				return err
			}
			return svc.NewWriter(tx).AddLabels(ctx, stored.Key, []label.Key{l.Key})
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return labels.NewWriter(tx).Delete(ctx, l.Key)
		})).To(Succeed())
		expected := stored
		expected.Labels = []label.Key{}
		got := sets()
		Expect(got[len(got)-1]).To(Equal(expected))
	})

	It("Should attach a set that holds the renamed status", func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Rename(ctx, stored.Key, "Valve")
		})).To(Succeed())
		renamed := stored
		renamed.Name = "Valve"
		Expect(sets()).To(Equal([]status.Status[any]{stored, renamed}))
	})

	It("Should attach a set that holds the labeled status", func(ctx SpecContext) {
		l := label.Label{Name: "Critical"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := labels.NewWriter(tx).Create(ctx, &l); err != nil {
				return err
			}
			return svc.NewWriter(tx).AddLabels(ctx, stored.Key, []label.Key{l.Key})
		})).To(Succeed())
		labeled := stored
		labeled.Labels = []label.Key{l.Key}
		Expect(sets()).To(Equal([]status.Status[any]{stored, labeled}))
	})

	It("Should replace the whole status when reduced", func() {
		next := stored
		next.Message = "Pump stopped"
		Expect(status.Reduce(
			stored,
			status.NewSetAction(status.SetPayload{Status: next}),
		)).To(Equal(next))
	})
})
