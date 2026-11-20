// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bind_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	tagRead  uint8 = 1
	tagWrite uint8 = 2
)

// tagged reads some labels and writes others, is placed in a range through one of two
// tagged place references, and is owned through one of two tagged use references.
type tagged struct {
	Key, Name           string
	Reads, Writes       []string
	In, InToo           string
	OwnedBy, OwnedByToo string
}

func (t tagged) GorpKey() string { return t.Key }

func (t *tagged) fields() *tagged { return t }

func (t tagged) EncodeOrc(w *orc.Writer) error {
	w.String(t.Key)
	w.String(t.Name)
	for _, keys := range [2][]string{t.Reads, t.Writes} {
		w.Uint32(uint32(len(keys)))
		for _, k := range keys {
			w.String(k)
		}
	}
	for _, s := range [4]string{t.In, t.InToo, t.OwnedBy, t.OwnedByToo} {
		w.String(s)
	}
	return nil
}

func (t *tagged) DecodeOrc(r *orc.Reader) (err error) {
	if t.Key, err = r.String(); err != nil {
		return err
	}
	if t.Name, err = r.String(); err != nil {
		return err
	}
	for _, keys := range [2]*[]string{&t.Reads, &t.Writes} {
		n, err := r.Uint32()
		if err != nil {
			return err
		}
		*keys = make([]string, n)
		for i := range *keys {
			if (*keys)[i], err = r.String(); err != nil {
				return err
			}
		}
	}
	for _, s := range [4]*string{&t.In, &t.InToo, &t.OwnedBy, &t.OwnedByToo} {
		if *s, err = r.String(); err != nil {
			return err
		}
	}
	return nil
}

// taggedUnmapped is a tagged record in a table with no resource type.
type taggedUnmapped struct{ tagged }

func fieldsOf(e any) *tagged { return e.(interface{ fields() *tagged }).fields() }

func targetsIn(table string, keys ...string) []gorp.Target {
	var out []gorp.Target
	for _, k := range keys {
		if k != "" {
			out = append(out, gorp.Target{Table: table, Key: k})
		}
	}
	return out
}

// openTagged opens a table of tagged records named name until the spec ends.
func openTagged[E gorp.Entry[string]](
	ctx context.Context,
	name string,
) *gorp.Table[string, E] {
	GinkgoHelper()
	var (
		lbl = string(resource.TypeLabel)
		rng = string(resource.TypeRange)
		usr = string(resource.TypeUser)
	)
	ref := func(
		name string,
		table string,
		kind gorp.LinkKind,
		onDelete gorp.OnDelete,
		tag uint8,
		targets func(t *tagged) []gorp.Target,
	) gorp.Reference[string, E] {
		return gorp.NewTargetReference(gorp.TargetReferenceConfig[string, E]{
			Name:     name,
			Tables:   []string{table},
			OnDelete: onDelete,
			Kind:     kind,
			Tag:      tag,
			Targets:  func(e *E) []gorp.Target { return targets(fieldsOf(e)) },
		})
	}
	return MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, E]{
		DB:   db,
		Name: name,
		References: []gorp.Reference[string, E]{
			ref(
				"reads",
				lbl,
				gorp.Use,
				gorp.Restrict,
				tagRead,
				func(t *tagged) []gorp.Target {
					return targetsIn(lbl, t.Reads...)
				},
			),
			ref(
				"writes",
				lbl,
				gorp.Use,
				gorp.Restrict,
				tagWrite,
				func(t *tagged) []gorp.Target {
					return targetsIn(lbl, t.Writes...)
				},
			),
			ref(
				"in",
				rng,
				gorp.Place,
				gorp.Cascade,
				tagRead,
				func(t *tagged) []gorp.Target {
					return targetsIn(rng, t.In)
				},
			),
			ref(
				"in_too",
				rng,
				gorp.Place,
				gorp.Cascade,
				tagWrite,
				func(t *tagged) []gorp.Target {
					return targetsIn(rng, t.InToo)
				},
			),
			ref(
				"owned_by",
				usr,
				gorp.Use,
				gorp.Cascade,
				tagRead,
				func(t *tagged) []gorp.Target {
					return targetsIn(usr, t.OwnedBy)
				},
			),
			ref(
				"owned_by_too",
				usr,
				gorp.Use,
				gorp.Cascade,
				tagWrite,
				func(t *tagged) []gorp.Target {
					return targetsIn(usr, t.OwnedByToo)
				},
			),
		},
	}))
}

// change returns a write that sets the record with key through f.
func change[E gorp.Entry[string]](key string, f func(t *tagged)) apply {
	return func(ctx context.Context, tx gorp.Tx) error {
		return gorp.NewUpdate[string, E]().
			Where(gorp.MatchKeys[string, E](key)).
			Change(func(_ gorp.Context, e E) E {
				f(fieldsOf(&e))
				return e
			}).
			Exec(ctx, tx)
	}
}

// tagCase is a stored record, a write that keeps its links, a write that changes only
// the tags of its links, and the access that both need.
type tagCase struct {
	kept, retagged apply
	needed         need
}

var _ = Describe("Write and reference tags", func() {
	var (
		views    *gorp.Table[string, tagged]
		unmapped *gorp.Table[string, taggedUnmapped]
	)
	BeforeEach(func(ctx SpecContext) {
		views = openTagged[tagged](ctx, string(resource.TypeFramer))
		unmapped = openTagged[taggedUnmapped](ctx, "bind_spec_tagged")
	})
	createView := func(ctx context.Context, t tagged) resource.ID {
		GinkgoHelper()
		Expect(views.NewCreate().Entry(&t).Exec(ctx, db)).To(Succeed())
		return resource.ID{Type: resource.TypeFramer, Key: t.Key}
	}
	rename := func(t *tagged) { t.Name = t.Name + "!" }

	DescribeTable(
		"Should need the same access for a change of tag alone as for a write that "+
			"keeps the link",
		func(ctx SpecContext, setup func(ctx context.Context) tagCase) {
			u := newUser(ctx)
			c := setup(ctx)
			// The retagged write changes tagged links, so Gorp reports them.
			rollback := errors.New("rollback")
			var changed []gorp.Link
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := c.retagged(ctx, tx); err != nil {
					return err
				}
				for _, ch := range gorp.ChangesOf(tx) {
					changed = append(append(changed, ch.Added...), ch.Removed...)
				}
				return rollback
			})).To(MatchError(rollback))
			Expect(changed).ToNot(BeEmpty())
			Expect(changed).To(HaveEach(HaveField("Tag", BeNumerically(">", 0))))
			Expect(write(ctx, u, c.kept)).
				To(denied(u, c.needed.action, c.needed.object))
			Expect(write(ctx, u, c.retagged)).
				To(denied(u, c.needed.action, c.needed.object))
			grant(ctx, u, c.needed.action, c.needed.object)
			Expect(write(ctx, u, c.kept)).To(Succeed())
			Expect(write(ctx, u, c.retagged)).To(Succeed())
		},
		Entry(
			"a use link that moves from a read to a write reference",
			func(ctx context.Context) tagCase {
				l := createLabel(ctx)
				id := createView(
					ctx,
					tagged{Key: "moves", Reads: []string{l.Key.String()}},
				)
				return tagCase{
					kept: change[tagged](id.Key, rename),
					retagged: change[tagged](id.Key, func(t *tagged) {
						t.Reads, t.Writes = nil, []string{l.Key.String()}
					}),
					needed: need{access.ActionUpdate, id},
				}
			},
		),
		Entry(
			"a holder that starts to write a label it reads",
			func(ctx context.Context) tagCase {
				l := createLabel(ctx)
				id := createView(
					ctx,
					tagged{Key: "starts", Reads: []string{l.Key.String()}},
				)
				return tagCase{
					kept: change[tagged](id.Key, rename),
					retagged: change[tagged](id.Key, func(t *tagged) {
						t.Writes = []string{l.Key.String()}
					}),
					needed: need{access.ActionUpdate, id},
				}
			},
		),
		Entry(
			"a holder that stops reading a label it also writes",
			func(ctx context.Context) tagCase {
				l := createLabel(ctx)
				k := l.Key.String()
				id := createView(ctx, tagged{
					Key:    "stops",
					Reads:  []string{k},
					Writes: []string{k},
				})
				return tagCase{
					kept:     change[tagged](id.Key, rename),
					retagged: change[tagged](id.Key, func(t *tagged) { t.Reads = nil }),
					needed:   need{access.ActionUpdate, id},
				}
			},
		),
		Entry(
			"a place link that moves to the other place reference",
			func(ctx context.Context) tagCase {
				r := createRange(ctx, nil)
				id := createView(ctx, tagged{Key: "placed", In: r.Key.String()})
				return tagCase{
					kept: change[tagged](id.Key, rename),
					retagged: change[tagged](id.Key, func(t *tagged) {
						t.In, t.InToo = "", r.Key.String()
					}),
					needed: need{access.ActionUpdate, id},
				}
			},
		),
		Entry(
			"a record with no resource type placed in its owner under a second tag",
			func(ctx context.Context) tagCase {
				r := createRange(ctx, nil)
				k := r.Key.String()
				Expect(unmapped.NewCreate().
					Entry(&taggedUnmapped{tagged{Key: "owned", In: k}}).
					Exec(ctx, db)).To(Succeed())
				return tagCase{
					kept: change[taggedUnmapped]("owned", rename),
					retagged: change[taggedUnmapped]("owned", func(t *tagged) {
						t.InToo = k
					}),
					needed: need{access.ActionUpdate, r.ResourceID()},
				}
			},
		),
		Entry(
			"a record with no resource type created in its owner under two tags",
			func(ctx context.Context) tagCase {
				r := createRange(ctx, nil)
				k := r.Key.String()
				return tagCase{
					kept: func(ctx context.Context, tx gorp.Tx) error {
						return unmapped.NewCreate().
							Entry(&taggedUnmapped{tagged{Key: "once", In: k}}).
							Exec(ctx, tx)
					},
					retagged: func(ctx context.Context, tx gorp.Tx) error {
						return unmapped.NewCreate().
							Entry(&taggedUnmapped{tagged{Key: "twice", In: k, InToo: k}}).
							Exec(ctx, tx)
					},
					needed: need{access.ActionUpdate, r.ResourceID()},
				}
			},
		),
		Entry(
			"a record with no resource type deleted from its owner under two tags",
			func(ctx context.Context) tagCase {
				r := createRange(ctx, nil)
				k := r.Key.String()
				Expect(unmapped.NewCreate().Entries(&[]taggedUnmapped{
					{tagged{Key: "gone_once", In: k}},
					{tagged{Key: "gone_twice", In: k, InToo: k}},
				}).Exec(ctx, db)).To(Succeed())
				del := func(key string) apply {
					return func(ctx context.Context, tx gorp.Tx) error {
						return unmapped.NewDelete().
							Where(gorp.MatchKeys[string, taggedUnmapped](key)).
							Exec(ctx, tx)
					}
				}
				return tagCase{
					kept:     del("gone_once"),
					retagged: del("gone_twice"),
					needed:   need{access.ActionUpdate, r.ResourceID()},
				}
			},
		),
	)

	Describe("Owned records", func() {
		var owning *bind.Service
		BeforeEach(func() {
			owning = MustSucceed(bind.NewService(bind.ServiceConfig{
				DB:    db,
				RBAC:  svc.RBAC,
				Held:  apiauth.HeldTables(),
				Owned: []string{string(resource.TypeFramer)},
			}))
		})
		writeOwned := func(ctx context.Context, u user.User, f apply) error {
			_, err := bind.Write(
				owning,
				func(ctx context.Context, tx gorp.Tx, _ struct{}) (struct{}, error) {
					return struct{}{}, f(ctx, tx)
				},
			)(AuthedCtx(ctx, u), struct{}{})
			return err
		}
		It(
			"Should let the owner move its owner link to another tag as it edits the "+
				"record",
			func(ctx SpecContext) {
				owner, other := newUser(ctx), newUser(ctx)
				id := createView(ctx, tagged{Key: "draft", OwnedBy: owner.Key.String()})
				kept := change[tagged](id.Key, rename)
				retagged := change[tagged](id.Key, func(t *tagged) {
					t.OwnedBy, t.OwnedByToo = "", owner.Key.String()
				})
				Expect(writeOwned(ctx, other, kept)).
					To(denied(other, access.ActionUpdate, id))
				Expect(writeOwned(ctx, other, retagged)).
					To(denied(other, access.ActionUpdate, id))
				Expect(writeOwned(ctx, owner, kept)).To(Succeed())
				Expect(writeOwned(ctx, owner, retagged)).To(Succeed())
				back := change[tagged](id.Key, func(t *tagged) {
					t.OwnedBy, t.OwnedByToo = owner.Key.String(), ""
				})
				Expect(writeOwned(ctx, owner, back)).To(Succeed())
			},
		)
	})
})
