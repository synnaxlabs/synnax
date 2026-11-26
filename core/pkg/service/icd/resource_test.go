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
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resource IDs", func() {
	It("Should build a resource ID of the ICD type", func() {
		key := uuid.New()
		Expect(icd.ResourceID(key)).To(Equal(resource.ID{
			Type: resource.TypeICD,
			Key:  key.String(),
		}))
	})

	It("Should map keys and ICDs to resource IDs", func() {
		a, b := uuid.New(), uuid.New()
		ids := []resource.ID{icd.ResourceID(a), icd.ResourceID(b)}
		Expect(icd.ResourceIDs([]icd.Key{a, b})).To(Equal(ids))
		Expect(icd.ResourceIDsFromICDs([]icd.ICD{{Key: a}, {Key: b}})).To(Equal(ids))
	})
})

var _ = Describe("Resources", func() {
	create := func(ctx SpecContext, name string) icd.ICD {
		GinkgoHelper()
		ic := icd.ICD{Name: name}
		Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
		return ic
	}

	It("Should name each ICD by its name", func(ctx SpecContext) {
		ic := create(ctx, "Engine bus")
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{ic.ResourceID()})).
			To(Equal([]resource.Resource{{ID: ic.ResourceID(), Name: "Engine bus"}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{{Type: resource.TypeICD, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid icd key "pump"`)),
		))
	})

	It("Should leave out an ID that names no ICD", func(ctx SpecContext) {
		ic := create(ctx, "Listed")
		missing := icd.ResourceID(uuid.New())
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, ic.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: ic.ResourceID(), Name: "Listed"}}))
	})

	It("Should list each ICD by its name", func(ctx SpecContext) {
		ic := create(ctx, "Listed")
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: ic.ResourceID(), Name: "Listed"},
		))
	})
})

// holder is a record that holds the key of an ICD, as a task config does.
type holder struct {
	Key string
	ICD icd.Key
}

func (h holder) GorpKey() string { return h.Key }

func (h holder) EncodeOrc(w *orc.Writer) error {
	w.String(h.Key)
	w.Write(h.ICD[:])
	return nil
}

func (h *holder) DecodeOrc(r *orc.Reader) (err error) {
	if h.Key, err = r.String(); err != nil {
		return err
	}
	_, err = r.Read(h.ICD[:])
	return err
}

// openHolders opens a table of holders whose reference refuses the delete of the ICD
// that a holder holds.
func openHolders(ctx context.Context, db *gorp.DB) *gorp.Table[string, holder] {
	GinkgoHelper()
	return MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, holder]{
		DB:   db,
		Name: "holder",
		References: []gorp.Reference[string, holder]{gorp.NewKeyReference(
			gorp.KeyReferenceConfig[string, holder, icd.Key]{
				Name:     "icd",
				Table:    "icd",
				OnDelete: gorp.Restrict,
				Kind:     gorp.Use,
				Targets:  func(h *holder) []icd.Key { return []icd.Key{h.ICD} },
			},
		)},
	}))
}

var _ = Describe("Uses", func() {
	var ic icd.ICD
	BeforeEach(func(ctx SpecContext) {
		ic = icd.ICD{Name: "Plant"}
		Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
	})

	It("Should return no records for an unused ICD", func(ctx SpecContext) {
		Expect(svc.Uses(ctx, tx, ic.Key)).To(BeEmpty())
	})

	It("Should return each record that uses the ICD", func(ctx SpecContext) {
		other := icd.ICD{Name: "Other"}
		Expect(svc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
		Expect(holders.NewCreate().Entries(&[]holder{
			{Key: "a", ICD: ic.Key},
			{Key: "b", ICD: ic.Key},
			{Key: "c", ICD: other.Key},
		}).Exec(ctx, tx)).To(Succeed())
		Expect(svc.Uses(ctx, tx, ic.Key)).To(ConsistOf(
			resource.ID{Type: "holder", Key: "a"},
			resource.ID{Type: "holder", Key: "b"},
		))
	})
})

var _ = Describe("Search", func() {
	create := func(ctx SpecContext, name string) resource.ID {
		GinkgoHelper()
		ic := icd.ICD{Name: name}
		Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
		return icd.ResourceID(ic.Key)
	}

	It("Should return the matching ICDs, closest match first", func(ctx SpecContext) {
		a := create(ctx, "turbopump test")
		b := create(ctx, "turbopump")
		q := search.Query[resource.ID]{Term: "turbopump"}
		Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
			To(Equal([]resource.ID{b, a}))
		q.Limit = 1
		Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
			To(Equal([]resource.ID{b}))
	})

	It("Should not match a staged ICD outside its transaction", func(ctx SpecContext) {
		create(ctx, "injector")
		q := search.Query[resource.ID]{Term: "injector"}
		Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).To(BeEmpty())
	})
})
