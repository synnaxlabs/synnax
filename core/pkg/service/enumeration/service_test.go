// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/orc"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

// user is a record that holds the key of an enumeration, as a symbol does.
type user struct {
	Key         string
	Enumeration enumeration.Key
}

func (u user) GorpKey() string { return u.Key }

func (u user) EncodeOrc(w *orc.Writer) error {
	w.String(u.Key)
	w.Write(u.Enumeration[:])
	return nil
}

func (u *user) DecodeOrc(r *orc.Reader) (err error) {
	if u.Key, err = r.String(); err != nil {
		return err
	}
	_, err = r.Read(u.Enumeration[:])
	return err
}

// openUsers opens a table of users whose reference refuses the delete of the
// enumeration that a user holds.
func openUsers(ctx context.Context, db *gorp.DB) *gorp.Table[string, user] {
	GinkgoHelper()
	return MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, user]{
		DB:   db,
		Name: "user",
		References: []gorp.Reference[string, user]{gorp.NewKeyReference(
			gorp.KeyReferenceConfig[string, user, enumeration.Key]{
				Name:     "enumeration",
				Table:    "enumeration",
				OnDelete: gorp.Restrict,
				Kind:     gorp.Use,
				Targets: func(u *user) []enumeration.Key {
					return []enumeration.Key{u.Enumeration}
				},
			},
		)},
	}))
}

var _ = Describe("Service", func() {
	var (
		db  *gorp.DB
		svc *enumeration.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		svc = MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}))
	})

	Describe("OpenService", func() {
		It("Should refuse a config with no DB", func(ctx SpecContext) {
			Expect(enumeration.OpenService(ctx, enumeration.ServiceConfig{})).
				Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("db: must be non-nil")),
				))
		})
	})

	Describe("Changes", func() {
		It("Should return the enumeration that a commit creates", func(
			ctx SpecContext,
		) {
			var names []string
			DeferCleanup(
				db.Observe().OnChange(func(ctx context.Context, c gorp.Commit) {
					for ch := range svc.Changes(c) {
						names = append(names, ch.Value.Name)
					}
				}),
			)
			e := enumeration.Enumeration{Name: "Breaker"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return createOne(ctx, svc.NewWriter(tx), &e)
			})).To(Succeed())
			Expect(names).To(Equal([]string{"Breaker"}))
		})
	})

	Describe("Uses", func() {
		var (
			users *gorp.Table[string, user]
			e     enumeration.Enumeration
			tx    gorp.Tx
		)
		BeforeEach(func(ctx SpecContext) {
			users = openUsers(ctx, db)
			tx = DeferClose(db.OpenTx())
			e = enumeration.Enumeration{Name: "Breaker"}
			Expect(createOne(ctx, svc.NewWriter(tx), &e)).To(Succeed())
		})

		It("Should return no records for an unused enumeration", func(
			ctx SpecContext,
		) {
			Expect(svc.Uses(ctx, tx, e.Key)).To(BeEmpty())
		})

		It("Should return each record that uses the enumeration", func(
			ctx SpecContext,
		) {
			Expect(users.NewCreate().Entries(&[]user{
				{Key: "a", Enumeration: e.Key},
				{Key: "b", Enumeration: e.Key},
			}).Exec(ctx, tx)).To(Succeed())
			Expect(svc.Uses(ctx, tx, e.Key)).To(ConsistOf(
				resource.ID{Type: "user", Key: "a"},
				resource.ID{Type: "user", Key: "b"},
			))
		})

		It("Should refuse the delete of an enumeration in use", func(ctx SpecContext) {
			Expect(users.NewCreate().
				Entry(&user{Key: "a", Enumeration: e.Key}).
				Exec(ctx, tx)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, e.Key)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"cannot delete enumeration %s because user a references it",
					e.Key,
				)),
			))
			Expect(svc.NewRetrieve().
				Where(enumeration.MatchKeys(e.Key)).
				Exists(ctx, tx)).To(BeTrue())
		})
	})
})
