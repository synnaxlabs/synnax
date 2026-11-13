// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/orc"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// rack stands in for a rack: a group can sit under one, and one can sit in a group.
type rack struct {
	Key   string     `json:"key"   msgpack:"key"`
	Group *group.Key `json:"group" msgpack:"group"`
}

func (r rack) GorpKey() string { return r.Key }

func (r rack) EncodeOrc(w *orc.Writer) error {
	w.String(r.Key)
	w.Bool(r.Group != nil)
	if r.Group != nil {
		w.Write(r.Group[:])
	}
	return nil
}

func (r *rack) DecodeOrc(rd *orc.Reader) error {
	var err error
	if r.Key, err = rd.String(); err != nil {
		return err
	}
	grouped, err := rd.Bool()
	if err != nil || !grouped {
		return err
	}
	r.Group = new(group.Key)
	_, err = rd.Read(r.Group[:])
	return err
}

// project stands in for a project, which a group can sit under.
type project struct {
	Key string `json:"key" msgpack:"key"`
}

func (p project) GorpKey() string { return p.Key }

func (p project) EncodeOrc(w *orc.Writer) error {
	w.String(p.Key)
	return nil
}

func (p *project) DecodeOrc(r *orc.Reader) (err error) {
	p.Key, err = r.String()
	return err
}

func rackID(key string) resource.ID {
	return resource.ID{Type: resource.TypeRack, Key: key}
}

var _ = Describe("Member", func() {
	var (
		db  *gorp.DB
		svc *group.Service
	)

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		svc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB: db,
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, rack]{
			DB:   db,
			Name: "rack",
			References: []gorp.Reference[string, rack]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, rack, group.Key]{
					Name:     "group",
					Table:    "group",
					OnDelete: gorp.Detach,
					Kind:     gorp.Place,
					Targets: func(r *rack) []group.Key {
						if r.Group == nil {
							return nil
						}
						return []group.Key{*r.Group}
					},
					Detach: func(r *rack, _ group.Key) { r.Group = nil },
				}),
			},
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, project]{
			DB:   db,
			Name: "project",
		}))
		Expect(gorp.NewCreate[string, rack]().
			Entries(&[]rack{{Key: "r1"}, {Key: "r2"}}).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.NewCreate[string, project]().
			Entry(&project{Key: "p1"}).
			Exec(ctx, db)).To(Succeed())
	})

	create := func(
		ctx context.Context,
		scope resource.Type,
		parent *resource.ID,
	) group.Group {
		GinkgoHelper()
		g := group.Group{Name: uuid.New().String(), Scope: scope, Parent: parent}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}

	place := func(ctx context.Context, key string, g group.Group) {
		GinkgoHelper()
		Expect(gorp.NewCreate[string, rack]().
			Entry(&rack{Key: key, Group: &g.Key}).
			Exec(ctx, db)).To(Succeed())
	}

	Describe("CheckMember", func() {
		It("Should allow a record in no group", func(ctx SpecContext) {
			Expect(group.CheckMember(ctx, db, nil, resource.TypeUser, nil)).
				To(Succeed())
		})

		It("Should allow a channel in a nested channel group", func(ctx SpecContext) {
			top := create(ctx, resource.TypeChannel, nil)
			nested := create(ctx, resource.TypeChannel, new(top.ResourceID()))
			Expect(group.CheckMember(
				ctx, db, &nested.Key, resource.TypeChannel, nil,
			)).To(Succeed())
		})

		It("Should reject a type that cannot sit in a group", func(ctx SpecContext) {
			g := create(ctx, resource.TypeChannel, nil)
			Expect(group.CheckMember(ctx, db, &g.Key, resource.TypeUser, nil)).
				To(MatchError("group: a user cannot sit in a group: validation error"))
		})

		It("Should reject a group that does not exist", func(ctx SpecContext) {
			key := uuid.New()
			Expect(group.CheckMember(ctx, db, &key, resource.TypeChannel, nil)).
				To(MatchError(
					"group: group " + key.String() + " does not exist: validation error",
				))
		})

		It("Should reject a group of another scope", func(ctx SpecContext) {
			g := create(ctx, resource.TypeDevice, nil)
			Expect(group.CheckMember(ctx, db, &g.Key, resource.TypeChannel, nil)).
				To(MatchError(
					"group: a channel cannot sit in a device group: validation error",
				))
		})

		It("Should allow a device in a group under its rack", func(ctx SpecContext) {
			top := create(ctx, resource.TypeDevice, new(rackID("r1")))
			nested := create(ctx, resource.TypeDevice, new(top.ResourceID()))
			Expect(group.CheckMember(
				ctx, db, &nested.Key, resource.TypeDevice, new(rackID("r1")),
			)).To(Succeed())
		})

		It(
			"Should reject a device in a group under another rack",
			func(ctx SpecContext) {
				g := create(ctx, resource.TypeDevice, new(rackID("r1")))
				Expect(group.CheckMember(
					ctx, db, &g.Key, resource.TypeDevice, new(rackID("r2")),
				)).To(MatchError(
					"group: group " + g.Name + " does not sit under rack r2: " +
						"validation error",
				))
			},
		)

		It("Should reject a device held by a device", func(ctx SpecContext) {
			g := create(ctx, resource.TypeDevice, new(rackID("r1")))
			holder := resource.ID{Type: resource.TypeDevice, Key: "chassis"}
			Expect(group.CheckMember(
				ctx, db, &g.Key, resource.TypeDevice, &holder,
			)).To(MatchError(
				"group: a device held by device chassis cannot sit in a group: " +
					"validation error",
			))
		})

		It("Should allow a rack in a top-level device group", func(ctx SpecContext) {
			g := create(ctx, resource.TypeDevice, nil)
			Expect(group.CheckMember(ctx, db, &g.Key, resource.TypeRack, nil)).
				To(Succeed())
		})

		It("Should reject a rack in a group under a rack", func(ctx SpecContext) {
			g := create(ctx, resource.TypeDevice, new(rackID("r1")))
			Expect(group.CheckMember(ctx, db, &g.Key, resource.TypeRack, nil)).
				To(MatchError(
					"group: a rack cannot sit in a group under rack r1: " +
						"validation error",
				))
		})
	})

	Describe("Create", func() {
		It("Should give a group under a rack the device scope", func(ctx SpecContext) {
			g := group.Group{Name: "under rack", Parent: new(rackID("r1"))}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
			Expect(g.Scope).To(Equal(resource.TypeDevice))
		})

		It(
			"Should give a group under a project the project scope",
			func(ctx SpecContext) {
				g := group.Group{
					Name:   "under project",
					Parent: &resource.ID{Type: resource.TypeProject, Key: "p1"},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &g)
				})).To(Succeed())
				Expect(g.Scope).To(Equal(resource.TypeProject))
			},
		)
	})

	Describe("SetParent", func() {
		It(
			"Should refuse to move a group that holds a rack under a new top parent",
			func(ctx SpecContext) {
				g := create(ctx, resource.TypeDevice, nil)
				nested := create(ctx, resource.TypeDevice, new(g.ResourceID()))
				place(ctx, "r3", nested)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).SetParent(ctx, new(rackID("r1")), g.Key)
				})).To(MatchError(
					"parent: cannot move group " + g.Name + " under a new top " +
						"parent because it holds rack r3: validation error",
				))
			},
		)

		It(
			"Should move a group that holds a rack under the same top parent",
			func(ctx SpecContext) {
				g := create(ctx, resource.TypeDevice, nil)
				dest := create(ctx, resource.TypeDevice, nil)
				place(ctx, "r3", g)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).
						SetParent(ctx, new(dest.ResourceID()), g.Key)
				})).To(Succeed())
			},
		)

		It(
			"Should move a group that holds no rack under a new top parent",
			func(ctx SpecContext) {
				g := create(ctx, resource.TypeDevice, nil)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).SetParent(ctx, new(rackID("r1")), g.Key)
				})).To(Succeed())
			},
		)
	})

	Describe("CreateObjects", func() {
		It("Should add each distinct group that new records join", func() {
			kept, a, b := uuid.New(), uuid.New(), uuid.New()
			stored := []rack{{Key: "kept", Group: &kept}}
			records := []rack{
				{Key: "kept", Group: &b},
				{Key: "r1", Group: &a},
				{Key: "r2"},
				{Key: "r3", Group: &a},
				{Key: "r4", Group: &b},
			}
			Expect(group.CreateObjects(
				records,
				stored,
				func(r rack) *group.Key { return r.Group },
			)).To(Equal([]resource.ID{group.ResourceID(a), group.ResourceID(b)}))
		})
	})
})
