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
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// member stands in for a record of the channel scope, which takes a name in a group.
type member struct {
	Key   string    `json:"key"   msgpack:"key"`
	Group group.Key `json:"group" msgpack:"group"`
	Name  string    `json:"name"  msgpack:"name"`
}

func (m member) GorpKey() string { return m.Key }

func (m member) EncodeOrc(w *orc.Writer) error {
	w.String(m.Key)
	w.Write(m.Group[:])
	w.String(m.Name)
	return nil
}

func (m *member) DecodeOrc(r *orc.Reader) error {
	var err error
	if m.Key, err = r.String(); err != nil {
		return err
	}
	if _, err = r.Read(m.Group[:]); err != nil {
		return err
	}
	m.Name, err = r.String()
	return err
}

var _ = Describe("NewMemberIndex", func() {
	var (
		db      *gorp.DB
		svc     *group.Service
		members *gorp.Table[string, member]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		svc = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		members = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, member]{
			DB:   db,
			Name: "member",
			Indexes: []gorp.Index[string, member]{group.NewMemberIndex(
				resource.TypeChannel,
				func(m *member) group.Sibling {
					return group.Sibling{Parent: m.Group, Name: m.Name}
				},
			)},
		}))
	})
	folder := func(ctx context.Context, name string, parent *group.Key) group.Group {
		GinkgoHelper()
		g := group.Group{Name: name, Scope: resource.TypeChannel}
		if parent != nil {
			g.Parent = new(group.ResourceID(*parent))
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	create := func(ctx context.Context, m member) error {
		return members.NewCreate().Entry(&m).Exec(ctx, db)
	}

	It("Should refuse a member whose name a folder of its scope holds in its group",
		func(ctx SpecContext) {
			stand := folder(ctx, "stand", nil)
			Expect(create(ctx, member{Key: "1", Name: "stand"})).
				To(MatchError(query.ErrUniqueViolation))
			Expect(create(ctx, member{Key: "2", Group: stand.Key, Name: "pt"})).
				To(Succeed())
		})

	It("Should take a member whose name a folder holds in another group", func(
		ctx SpecContext,
	) {
		plant := folder(ctx, "plant", nil)
		folder(ctx, "stand", &plant.Key)
		Expect(create(ctx, member{Key: "1", Name: "stand"})).To(Succeed())
	})

	It("Should refuse a second member of one name in one group", func(ctx SpecContext) {
		Expect(create(ctx, member{Key: "1", Group: uuid.Nil(), Name: "pt"})).
			To(Succeed())
		Expect(create(ctx, member{Key: "2", Group: uuid.Nil(), Name: "pt"})).
			To(MatchError(query.ErrUniqueViolation))
	})
})
