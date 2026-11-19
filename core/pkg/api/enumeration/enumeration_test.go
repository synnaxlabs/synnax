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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apienumeration "github.com/synnaxlabs/synnax/pkg/api/enumeration"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// deleteAfter deletes the enumeration with the given key after the spec.
func deleteAfter(key enumeration.Key) {
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Enumeration.NewWriter(tx).Delete(ctx, key)
		})).To(Succeed())
	})
}

func createEnumeration(ctx SpecContext, name string) enumeration.Enumeration {
	GinkgoHelper()
	e := enumeration.Enumeration{
		Name:    name,
		Entries: []enumeration.Entry{{Key: "off", Value: 0, Name: "Off"}},
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return createOne(ctx, svc.Enumeration.NewWriter(tx), &e)
	})).To(Succeed())
	deleteAfter(e.Key)
	return e
}

func retrieveEnumeration(ctx SpecContext, key enumeration.Key) enumeration.Enumeration {
	GinkgoHelper()
	var e enumeration.Enumeration
	Expect(svc.Enumeration.NewRetrieve().
		Where(enumeration.MatchKeys(key)).
		Entry(&e).
		Exec(ctx, db)).To(Succeed())
	return e
}

func enumerationExists(ctx SpecContext, key enumeration.Key) bool {
	GinkgoHelper()
	return MustSucceed(
		svc.Enumeration.NewRetrieve().Where(enumeration.MatchKeys(key)).Exists(ctx, db),
	)
}

var anyEnumeration = resource.ID{Type: resource.TypeEnumeration}

var _ = Describe("Create", func() {
	It(
		"Should create the enumerations and stamp the dispatch key onto the broadcast",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, access.ActionCreate, u.ResourceID(), anyEnumeration)
			rec := &Recorder[enumeration.Key, enumeration.Action]{}
			DeferCleanup(svc.Enumeration.Actions().OnChange(rec.Record))
			res := MustSucceed(create(AuthedCtx(ctx, u), apienumeration.CreateRequest{
				Enumerations: []enumeration.Enumeration{{Name: "Breaker"}},
				DispatchKey:  "dk-create",
			}))
			Expect(res.Enumerations).To(HaveLen(1))
			deleteAfter(res.Enumerations[0].Key)
			Expect(retrieveEnumeration(ctx, res.Enumerations[0].Key)).
				To(Equal(res.Enumerations[0]))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		},
	)

	It("Should refuse a create without create on the enumeration", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		e := enumeration.Enumeration{Key: uuid.New(), Name: "Breaker"}
		Expect(create(AuthedCtx(ctx, u), apienumeration.CreateRequest{
			Enumerations: []enumeration.Enumeration{e},
		})).Error().To(denied(u, access.ActionCreate, e.ResourceID()))
		Expect(enumerationExists(ctx, e.Key)).To(BeFalse())
	})

	It("Should refuse two entries with one value", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionCreate, u.ResourceID(), anyEnumeration)
		Expect(create(AuthedCtx(ctx, u), apienumeration.CreateRequest{
			Enumerations: []enumeration.Enumeration{{
				Name: "Breaker",
				Entries: []enumeration.Entry{
					{Key: "a", Value: 1, Name: "A"},
					{Key: "b", Value: 1, Name: "B"},
				},
			}},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"entries.1.value: another entry has the value 1",
			)),
		))
	})
})

var _ = Describe("Retrieve", func() {
	It("Should leave out the enumerations that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		granted := createEnumeration(ctx, "Breaker")
		createEnumeration(ctx, "Damper")
		grant(ctx, access.ActionRetrieve, u.ResourceID(), granted.ResourceID())
		res := MustSucceed(
			retrieve(AuthedCtx(ctx, u), apienumeration.RetrieveRequest{}),
		)
		Expect(res.Enumerations).To(Equal([]enumeration.Enumeration{granted}))
	})

	It("Should refuse a key that the subject cannot retrieve", func(ctx SpecContext) {
		u := createUser(ctx)
		e := createEnumeration(ctx, "Breaker")
		Expect(retrieve(AuthedCtx(ctx, u), apienumeration.RetrieveRequest{
			Where: &enumeration.Where{Key: &filter.Set[enumeration.Key]{Eq: &e.Key}},
		})).Error().To(denied(u, access.ActionRetrieve, e.ResourceID()))
	})

	It("Should search the enumerations by name, closest first", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		states := createEnumeration(ctx, "Breaker states")
		breaker := createEnumeration(ctx, "Breaker")
		createEnumeration(ctx, "Damper")
		grant(ctx, access.ActionRetrieve, u.ResourceID(), anyEnumeration)
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apienumeration.RetrieveRequest{
			SearchTerm: "breaker",
		}))
		Expect(res.Enumerations).To(Equal([]enumeration.Enumeration{breaker, states}))
	})

	It("Should sort the enumerations by name", func(ctx SpecContext) {
		u := createUser(ctx)
		mine := []enumeration.Enumeration{
			createEnumeration(ctx, "Damper"),
			createEnumeration(ctx, "alarm"),
			createEnumeration(ctx, "Breaker"),
		}
		grant(ctx, access.ActionRetrieve, u.ResourceID(), anyEnumeration)
		keys := lo.Map(mine, func(e enumeration.Enumeration, _ int) enumeration.Key {
			return e.Key
		})
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apienumeration.RetrieveRequest{
			Where: &enumeration.Where{Key: &filter.Set[enumeration.Key]{In: keys}},
			Order: &apienumeration.Order{Field: apienumeration.OrderFieldName},
		}))
		Expect(lo.Map(res.Enumerations, func(e enumeration.Enumeration, _ int) string {
			return e.Name
		})).To(Equal([]string{"alarm", "Breaker", "Damper"}))
	})
})

var _ = Describe("Dispatch", func() {
	rename := []enumeration.Action{enumeration.NewSetAction(enumeration.SetPayload{
		Path:  []string{"name"},
		Value: "Damper",
	})}
	It("Should apply the actions when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		e := createEnumeration(ctx, "Breaker")
		grant(ctx, access.ActionUpdate, u.ResourceID(), e.ResourceID())
		MustSucceed(dispatch(
			AuthedCtx(ctx, u),
			apienumeration.DispatchRequest{
				Key:         e.Key,
				DispatchKey: "dk",
				Actions:     rename,
			},
		))
		Expect(retrieveEnumeration(ctx, e.Key).Name).To(Equal("Damper"))
	})
	It("Should refuse the actions when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		e := createEnumeration(ctx, "Breaker")
		Expect(dispatch(
			AuthedCtx(ctx, u),
			apienumeration.DispatchRequest{
				Key:         e.Key,
				DispatchKey: "dk",
				Actions:     rename,
			},
		)).Error().To(denied(u, access.ActionUpdate, e.ResourceID()))
		Expect(retrieveEnumeration(ctx, e.Key).Name).To(Equal("Breaker"))
	})
	It("Should return the error of the dispatch", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.ResourceID(), enumeration.ResourceID(key))
		Expect(dispatch(
			AuthedCtx(ctx, u),
			apienumeration.DispatchRequest{
				Key:         key,
				DispatchKey: "dk",
				Actions:     rename,
			},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Delete", func() {
	It("Should delete the enumerations when delete is granted", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		e := createEnumeration(ctx, "Breaker")
		grant(ctx, access.ActionDelete, u.ResourceID(), e.ResourceID())
		MustSucceed(del(AuthedCtx(ctx, u), apienumeration.DeleteRequest{
			Keys: []enumeration.Key{e.Key},
		}))
		Expect(enumerationExists(ctx, e.Key)).To(BeFalse())
	})
	It("Should refuse the delete without delete on the enumeration", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		e := createEnumeration(ctx, "Breaker")
		Expect(del(AuthedCtx(ctx, u), apienumeration.DeleteRequest{
			Keys: []enumeration.Key{e.Key},
		})).Error().To(denied(u, access.ActionDelete, e.ResourceID()))
		Expect(enumerationExists(ctx, e.Key)).To(BeTrue())
	})
})

var _ = Describe("RetrieveBuiltins", func() {
	It("Should return the shipped values of the built-in enumerations", func(
		ctx SpecContext,
	) {
		res := MustSucceed(apiSvc.RetrieveBuiltins(ctx, struct{}{}))
		Expect(res.Enumerations).To(Equal(enumeration.Builtins()))
		Expect(res.Enumerations).To(HaveLen(5))
	})
})

// holder is a record that holds the key of an enumeration, as a symbol does. Its table
// takes the name of a resource type that no Core table uses, so a policy can name it.
type holder struct {
	Key         string
	Enumeration enumeration.Key
}

func (h holder) GorpKey() string { return h.Key }

func (h holder) EncodeOrc(w *orc.Writer) error {
	w.String(h.Key)
	w.Write(h.Enumeration[:])
	return nil
}

func (h *holder) DecodeOrc(r *orc.Reader) (err error) {
	if h.Key, err = r.String(); err != nil {
		return err
	}
	_, err = r.Read(h.Enumeration[:])
	return err
}

// createHolders creates a holder of the enumeration for each key, and deletes them
// after the spec.
func createHolders(ctx SpecContext, e enumeration.Key, keys ...string) {
	GinkgoHelper()
	holders := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, holder]{
		DB:   db,
		Name: string(resource.TypeModbusServer),
		References: []gorp.Reference[string, holder]{gorp.NewKeyReference(
			gorp.KeyReferenceConfig[string, holder, enumeration.Key]{
				Name:     "enumeration",
				Table:    "enumeration",
				OnDelete: gorp.Restrict,
				Kind:     gorp.Use,
				Targets: func(h *holder) []enumeration.Key {
					return []enumeration.Key{h.Enumeration}
				},
				Retarget: func(h *holder, _, to enumeration.Key) { h.Enumeration = to },
			},
		)},
	}))
	entries := make([]holder, len(keys))
	for i, k := range keys {
		entries[i] = holder{Key: k, Enumeration: e}
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return holders.NewCreate().Entries(&entries).Exec(ctx, tx)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return holders.NewDelete().
				Where(gorp.MatchKeys[string, holder](keys...)).
				Exec(ctx, tx)
		})).To(Succeed())
	})
}

// holderID returns the resource ID of the holder with the given key.
func holderID(key string) resource.ID {
	return resource.ID{Type: resource.TypeModbusServer, Key: key}
}

// retrieveHolder returns the enumeration that the holder with the given key uses.
func retrieveHolder(ctx SpecContext, key string) enumeration.Key {
	GinkgoHelper()
	var h holder
	Expect(gorp.NewRetrieve[string, holder]().
		Where(gorp.MatchKeys[string, holder](key)).
		Entry(&h).
		Exec(ctx, db)).To(Succeed())
	return h.Enumeration
}

var _ = Describe("Replace", func() {
	var from, to enumeration.Enumeration
	BeforeEach(func(ctx SpecContext) {
		from = createEnumeration(ctx, "Breaker")
		to = createEnumeration(ctx, "Valve")
		createHolders(ctx, from.Key, "a", "b")
	})

	It("Should move each user to the other enumeration and delete the first", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.ActionDelete, u.ResourceID(), from.ResourceID())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), to.ResourceID())
		grant(ctx, access.ActionUpdate, u.ResourceID(), holderID("a"), holderID("b"))
		MustSucceed(replace(AuthedCtx(ctx, u), apienumeration.ReplaceRequest{
			From: from.Key,
			To:   to.Key,
		}))
		Expect(retrieveHolder(ctx, "a")).To(Equal(to.Key))
		Expect(retrieveHolder(ctx, "b")).To(Equal(to.Key))
		Expect(enumerationExists(ctx, from.Key)).To(BeFalse())
	})

	It("Should refuse the replace without update on a user", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionDelete, u.ResourceID(), from.ResourceID())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), to.ResourceID())
		grant(ctx, access.ActionUpdate, u.ResourceID(), holderID("a"))
		Expect(replace(AuthedCtx(ctx, u), apienumeration.ReplaceRequest{
			From: from.Key,
			To:   to.Key,
		})).Error().To(denied(u, access.ActionUpdate, holderID("b")))
		Expect(retrieveHolder(ctx, "a")).To(Equal(from.Key))
		Expect(enumerationExists(ctx, from.Key)).To(BeTrue())
	})

	It("Should refuse the replace without retrieve on the replacement", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.ActionDelete, u.ResourceID(), from.ResourceID())
		grant(ctx, access.ActionUpdate, u.ResourceID(), holderID("a"), holderID("b"))
		Expect(replace(AuthedCtx(ctx, u), apienumeration.ReplaceRequest{
			From: from.Key,
			To:   to.Key,
		})).Error().To(denied(u, access.ActionRetrieve, to.ResourceID()))
		Expect(enumerationExists(ctx, from.Key)).To(BeTrue())
	})

	It("Should refuse the replace without delete on the enumeration", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), to.ResourceID())
		grant(ctx, access.ActionUpdate, u.ResourceID(), holderID("a"), holderID("b"))
		Expect(replace(AuthedCtx(ctx, u), apienumeration.ReplaceRequest{
			From: from.Key,
			To:   to.Key,
		})).Error().To(denied(u, access.ActionDelete, from.ResourceID()))
		Expect(retrieveHolder(ctx, "a")).To(Equal(from.Key))
	})
})

var _ = Describe("Retrieve uses", func() {
	var e enumeration.Enumeration
	BeforeEach(func(ctx SpecContext) {
		e = createEnumeration(ctx, "Breaker")
		createHolders(ctx, e.Key, "a", "b")
	})

	// retrieveUses returns the enumeration with key as u sees it, with its uses.
	retrieveUses := func(
		ctx SpecContext,
		u user.User,
		key enumeration.Key,
	) enumeration.Enumeration {
		GinkgoHelper()
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apienumeration.RetrieveRequest{
			Where: &enumeration.Where{
				Key: &filter.Set[enumeration.Key]{Eq: &key},
			},
			IncludeUses: true,
		}))
		Expect(res.Enumerations).To(HaveLen(1))
		return res.Enumerations[0]
	}

	It("Should return the uses that the subject can retrieve", func(ctx SpecContext) {
		u := createUser(ctx)
		a := holderID("a")
		grant(ctx, access.ActionRetrieve, u.ResourceID(), e.ResourceID(), a)
		Expect(retrieveUses(ctx, u, e.Key).Uses).To(Equal([]resource.ID{a}))
	})

	It("Should return empty uses for an enumeration that nothing uses", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		unused := createEnumeration(ctx, "Damper")
		grant(ctx, access.ActionRetrieve, u.ResourceID(), unused.ResourceID())
		Expect(retrieveUses(ctx, u, unused.Key).Uses).To(Equal([]resource.ID{}))
	})

	It("Should leave out uses that the request does not ask for", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), e.ResourceID(), holderID("a"))
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apienumeration.RetrieveRequest{
			Where: &enumeration.Where{Key: &filter.Set[enumeration.Key]{Eq: &e.Key}},
		}))
		Expect(res.Enumerations).To(HaveLen(1))
		Expect(res.Enumerations[0].Uses).To(BeNil())
	})
})
