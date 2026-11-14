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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	xtypes "github.com/synnaxlabs/x/types"
)

// copyRequest is a request that copies its sources.
type copyRequest struct{ sources []resource.ID }

var _ bind.Copy = copyRequest{}

func (r copyRequest) Sources() []resource.ID { return r.sources }

// unmapped is a record whose table has no resource type.
type unmapped struct{ Key string }

func (u unmapped) GorpKey() string { return u.Key }

func (u unmapped) EncodeOrc(w *orc.Writer) error {
	w.String(u.Key)
	return nil
}

func (u *unmapped) DecodeOrc(r *orc.Reader) (err error) {
	u.Key, err = r.String()
	return err
}

// taskConfig is a config record placed in the task that shares its key.
type taskConfig struct{ Key string }

func (taskConfig) CustomTypeName() string { return config.TableName("bind_spec") }

func (c taskConfig) GorpKey() string { return c.Key }

func (c taskConfig) EncodeOrc(w *orc.Writer) error {
	w.String(c.Key)
	return nil
}

func (c *taskConfig) DecodeOrc(r *orc.Reader) (err error) {
	c.Key, err = r.String()
	return err
}

// movedTaskConfig is a config record of the task type that a task changes to.
type movedTaskConfig struct{ taskConfig }

func (movedTaskConfig) CustomTypeName() string { return config.TableName("bind_moved") }

// placedTwice is a record with no resource type that is placed in two ranges.
type placedTwice struct{ Key, First, Second string }

func (p placedTwice) GorpKey() string { return p.Key }

func (p placedTwice) EncodeOrc(w *orc.Writer) error {
	w.String(p.Key)
	w.String(p.First)
	w.String(p.Second)
	return nil
}

func (p *placedTwice) DecodeOrc(r *orc.Reader) (err error) {
	if p.Key, err = r.String(); err != nil {
		return err
	}
	if p.First, err = r.String(); err != nil {
		return err
	}
	p.Second, err = r.String()
	return err
}

// openConfigs opens a table of config records, each placed in the task that shares its
// key, until the spec ends.
func openConfigs[E gorp.Entry[string]](ctx context.Context) *gorp.Table[string, E] {
	GinkgoHelper()
	task := string(resource.TypeTask)
	return MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, E]{
		DB:   db,
		Name: xtypes.Name[E](),
		References: []gorp.Reference[string, E]{
			gorp.NewTargetReference(gorp.TargetReferenceConfig[string, E]{
				Name:     "key",
				Tables:   []string{task},
				OnDelete: gorp.Cascade,
				Kind:     gorp.Place,
				Targets: func(c *E) []gorp.Target {
					return []gorp.Target{gorp.NewTarget(task, (*c).GorpKey())}
				},
			}),
		},
	}))
}

// apply is the write of a spec.
type apply func(ctx context.Context, tx gorp.Tx) error

// newUser creates a user that can do nothing until a spec grants it access.
func newUser(ctx context.Context) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// grant gives u the action on the objects.
func grant(
	ctx context.Context,
	u user.User,
	action access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	MustSucceed(rbactestutil.Grant(
		ctx,
		db,
		svc.RBAC,
		svc.User,
		u.Key,
		[]access.Action{action},
		objects...,
	))
}

// write runs f through bind.Write as u, with a request that copies sources.
func write(ctx context.Context, u user.User, f apply, sources ...resource.ID) error {
	_, err := bind.Write(
		binder,
		func(ctx context.Context, tx gorp.Tx, _ copyRequest) (struct{}, error) {
			return struct{}{}, f(ctx, tx)
		},
	)(AuthedCtx(ctx, u), copyRequest{sources: sources})
	return err
}

// returned is a response that returns records.
type returned struct{ records []resource.ID }

func (r returned) Returned() []resource.ID { return r.records }

// writeReturning runs f through bind.Write as u, with a response that returns records.
func writeReturning(
	ctx context.Context,
	u user.User,
	f apply,
	records ...resource.ID,
) error {
	_, err := bind.Write(
		binder,
		func(ctx context.Context, tx gorp.Tx, _ struct{}) (returned, error) {
			if err := f(ctx, tx); err != nil {
				return returned{}, err
			}
			return returned{records: records}, nil
		},
	)(AuthedCtx(ctx, u), struct{}{})
	return err
}

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}

func createLabel(ctx context.Context) label.Label {
	GinkgoHelper()
	l := label.Label{Name: "label"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Label.NewWriter(tx).Create(ctx, &l)
	})).To(Succeed())
	return l
}

func newRange(parent *ranger.Key, labels ...label.Key) ranger.Range {
	return ranger.Range{
		Key:  uuid.New(),
		Name: "range",
		TimeRange: telem.TimeRange{
			Start: telem.TimeStamp(5 * telem.Second),
			End:   telem.TimeStamp(10 * telem.Second),
		},
		Parent: parent,
		Labels: labels,
	}
}

func createRange(
	ctx context.Context,
	parent *ranger.Key,
	labels ...label.Key,
) ranger.Range {
	GinkgoHelper()
	r := newRange(parent, labels...)
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Ranger.NewWriter(tx).Create(ctx, &r)
	})).To(Succeed())
	return r
}

func createTask(ctx context.Context) task.Task {
	GinkgoHelper()
	t := task.Task{
		Type: pagerduty.AlertTaskType,
		Rack: svc.Rack.EmbeddedKey,
		Name: "task",
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Task.NewWriter(tx).Create(ctx, &t)
	})).To(Succeed())
	return t
}

func setPair(ctx context.Context, tx gorp.Tx, rng ranger.Key, value string) error {
	return svc.KV.NewWriter(tx).
		SetMany(ctx, []kv.Pair{{Range: rng, Key: "k", Value: value}})
}

func labelExists(ctx context.Context, key label.Key) bool {
	GinkgoHelper()
	err := svc.Label.NewRetrieve().
		Where(label.MatchKeys(key)).
		Entry(&label.Label{}).
		Exec(ctx, db)
	Expect(errors.Skip(err, query.ErrNotFound)).To(Succeed())
	return err == nil
}

// need is an action on an object.
type need struct {
	action access.Action
	object resource.ID
}

// rule is a write, the sources it copies, the access it needs besides needed, and the
// access that the rule under test requires.
type rule struct {
	write   apply
	sources []resource.ID
	granted []need
	needed  need
}

var _ = Describe("Write", func() {
	DescribeTable(
		"Should deny a write and name the record without the access a rule requires",
		func(ctx SpecContext, setup func(ctx context.Context) rule) {
			u := newUser(ctx)
			r := setup(ctx)
			for _, g := range r.granted {
				grant(ctx, u, g.action, g.object)
			}
			Expect(write(ctx, u, r.write, r.sources...)).
				To(denied(u, r.needed.action, r.needed.object))
			grant(ctx, u, r.needed.action, r.needed.object)
			Expect(write(ctx, u, r.write, r.sources...)).To(Succeed())
		},
		Entry("create on a new record", func(ctx context.Context) rule {
			key := uuid.New()
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					l := label.Label{Key: key, Name: "new"}
					return svc.Label.NewWriter(tx).Create(ctx, &l)
				},
				needed: need{access.ActionCreate, label.ResourceID(key)},
			}
		}),
		Entry("update on a changed record", func(ctx context.Context) rule {
			stored := createLabel(ctx)
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					return svc.Label.NewWriter(tx).Apply(
						ctx,
						stored.Key,
						label.NewSetAction(
							label.SetPayload{Path: []string{"name"}, Value: "changed"},
						),
					)
				},
				needed: need{access.ActionUpdate, stored.ResourceID()},
			}
		}),
		Entry("delete on a deleted record", func(ctx context.Context) rule {
			l := createLabel(ctx)
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					return svc.Label.NewWriter(tx).Delete(ctx, l.Key)
				},
				needed: need{access.ActionDelete, l.ResourceID()},
			}
		}),
		Entry("delete on a cascaded record", func(ctx context.Context) rule {
			parent := createRange(ctx, nil)
			child := createRange(ctx, &parent.Key)
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					return svc.Ranger.NewWriter(tx).Delete(ctx, parent.Key)
				},
				granted: []need{{access.ActionDelete, parent.ResourceID()}},
				needed:  need{access.ActionDelete, child.ResourceID()},
			}
		}),
		Entry(
			"update on the place target of a deleted record",
			func(ctx context.Context) rule {
				parent := createRange(ctx, nil)
				child := createRange(ctx, &parent.Key)
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						return svc.Ranger.NewWriter(tx).Delete(ctx, child.Key)
					},
					granted: []need{{access.ActionDelete, child.ResourceID()}},
					needed:  need{access.ActionUpdate, parent.ResourceID()},
				}
			},
		),
		Entry("update on a detached record", func(ctx context.Context) rule {
			l := createLabel(ctx)
			holder := createRange(ctx, nil, l.Key)
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					return svc.Label.NewWriter(tx).Delete(ctx, l.Key)
				},
				granted: []need{{access.ActionDelete, l.ResourceID()}},
				needed:  need{access.ActionUpdate, holder.ResourceID()},
			}
		}),
		Entry(
			"update on the target of a place link added",
			func(ctx context.Context) rule {
				parent := createRange(ctx, nil)
				child := newRange(&parent.Key)
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						r := child
						return svc.Ranger.NewWriter(tx).Create(ctx, &r)
					},
					granted: []need{{access.ActionCreate, child.ResourceID()}},
					needed:  need{access.ActionUpdate, parent.ResourceID()},
				}
			},
		),
		Entry(
			"update on the target of a place link removed",
			func(ctx context.Context) rule {
				from := createRange(ctx, nil)
				to := createRange(ctx, nil)
				child := createRange(ctx, &from.Key)
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						return svc.Ranger.NewWriter(tx).Apply(
							ctx,
							child.Key,
							ranger.NewSetAction(ranger.SetPayload{
								Path:  []string{"parent"},
								Value: to.Key,
							}),
						)
					},
					granted: []need{
						{access.ActionUpdate, child.ResourceID()},
						{access.ActionUpdate, to.ResourceID()},
					},
					needed: need{access.ActionUpdate, from.ResourceID()},
				}
			},
		),
		Entry("retrieve on a source", func(ctx context.Context) rule {
			source := createLabel(ctx)
			key := uuid.New()
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					l := label.Label{Key: key, Name: source.Name}
					return svc.Label.NewWriter(tx).Create(ctx, &l)
				},
				sources: []resource.ID{source.ResourceID()},
				granted: []need{{access.ActionCreate, label.ResourceID(key)}},
				needed:  need{access.ActionRetrieve, source.ResourceID()},
			}
		}),
		Entry(
			"update on the task of a changed config record",
			func(ctx context.Context) rule {
				t := createTask(ctx)
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						_, err := svc.Task.NewWriter(tx).Apply(
							ctx,
							t.Key,
							task.NewSetAction(task.SetPayload{
								Path:  []string{"config", "auto_start"},
								Value: true,
							}),
						)
						return err
					},
					needed: need{access.ActionUpdate, t.ResourceID()},
				}
			},
		),
		Entry(
			"retrieve on the target of a use link added",
			func(ctx context.Context) rule {
				l := createLabel(ctx)
				r := createRange(ctx, nil)
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						return svc.Ranger.NewWriter(tx).
							AddLabels(ctx, r.Key, []label.Key{l.Key})
					},
					granted: []need{{access.ActionUpdate, r.ResourceID()}},
					needed:  need{access.ActionRetrieve, l.ResourceID()},
				}
			},
		),
		Entry(
			"update on the range of a new range key-value pair",
			func(ctx context.Context) rule {
				r := createRange(ctx, nil)
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						return setPair(ctx, tx, r.Key, "new")
					},
					needed: need{access.ActionUpdate, r.ResourceID()},
				}
			},
		),
		Entry(
			"update on the range of a new range key-value pair",
			func(ctx context.Context) rule {
				r := createRange(ctx, nil)
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						return setPair(ctx, tx, r.Key, "new")
					},
					needed: need{access.ActionUpdate, r.ResourceID()},
				}
			},
		),
		Entry(
			"update on the range of a changed range key-value pair",
			func(ctx context.Context) rule {
				r := createRange(ctx, nil)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return setPair(ctx, tx, r.Key, "stored")
				})).To(Succeed())
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						return setPair(ctx, tx, r.Key, "changed")
					},
					needed: need{access.ActionUpdate, r.ResourceID()},
				}
			},
		),
		Entry(
			"update on the range of a deleted range key-value pair",
			func(ctx context.Context) rule {
				r := createRange(ctx, nil)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return setPair(ctx, tx, r.Key, "stored")
				})).To(Succeed())
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						return svc.KV.NewWriter(tx).Delete(ctx, r.Key, "k")
					},
					needed: need{access.ActionUpdate, r.ResourceID()},
				}
			},
		),
		Entry("create on the user of a new API key", func(ctx context.Context) rule {
			other := newUser(ctx)
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					_, _, err := svc.APIKey.NewWriter(tx).
						Create(ctx, other.ResourceID(), "key")
					return err
				},
				needed: need{access.ActionCreate, other.ResourceID()},
			}
		}),
		Entry(
			"delete on the user of a deleted API key",
			func(ctx context.Context) rule {
				other := newUser(ctx)
				var k apikey.APIKey
				Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
					k, _, err = svc.APIKey.NewWriter(tx).
						Create(ctx, other.ResourceID(), "key")
					return err
				})).To(Succeed())
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						return svc.APIKey.NewWriter(tx).Delete(ctx, k.Key)
					},
					needed: need{access.ActionDelete, other.ResourceID()},
				}
			},
		),
		Entry("create on the user of a new password", func(ctx context.Context) rule {
			other := newUser(ctx)
			h := MustSucceed(svc.Password.Hash("secret"))
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					return svc.Password.NewWriter(tx).
						Set(ctx, other.ResourceID(), other.Name, h)
				},
				needed: need{access.ActionCreate, other.ResourceID()},
			}
		}),
		Entry(
			"update on the user of a changed password",
			func(ctx context.Context) rule {
				other := newUser(ctx)
				stored := MustSucceed(svc.Password.Hash("stored"))
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Password.NewWriter(tx).
						Set(ctx, other.ResourceID(), other.Name, stored)
				})).To(Succeed())
				changed := MustSucceed(svc.Password.Hash("changed"))
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						return svc.Password.NewWriter(tx).Apply(
							ctx,
							other.ResourceID(),
							password.NewSetHashAction(changed),
						)
					},
					needed: need{access.ActionUpdate, other.ResourceID()},
				}
			},
		),
		Entry("create on a new child range", func(ctx context.Context) rule {
			parent := createRange(ctx, nil)
			child := newRange(&parent.Key)
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					return svc.Ranger.NewWriter(tx).Create(ctx, &child)
				},
				granted: []need{{access.ActionUpdate, parent.ResourceID()}},
				needed:  need{access.ActionCreate, child.ResourceID()},
			}
		}),
		Entry("update on the task of a new task config record", func(
			ctx context.Context,
		) rule {
			configs := openConfigs[taskConfig](ctx)
			t := createTask(ctx)
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					return configs.NewCreate().
						Entry(&taskConfig{Key: t.Key.String()}).
						Exec(ctx, tx)
				},
				needed: need{access.ActionUpdate, task.ResourceID(t.Key)},
			}
		}),
		Entry("update on the task of a changed task config record", func(
			ctx context.Context,
		) rule {
			configs := openConfigs[taskConfig](ctx)
			t := createTask(ctx)
			c := taskConfig{Key: t.Key.String()}
			Expect(configs.NewCreate().Entry(&c).Exec(ctx, db)).To(Succeed())
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					return configs.NewCreate().Entry(&c).Exec(ctx, tx)
				},
				needed: need{access.ActionUpdate, task.ResourceID(t.Key)},
			}
		}),
		Entry(
			"update on the task of a task config record that moves to another table",
			func(ctx context.Context) rule {
				configs := openConfigs[taskConfig](ctx)
				moved := openConfigs[movedTaskConfig](ctx)
				t := createTask(ctx)
				c := taskConfig{Key: t.Key.String()}
				Expect(configs.NewCreate().Entry(&c).Exec(ctx, db)).To(Succeed())
				return rule{
					write: func(ctx context.Context, tx gorp.Tx) error {
						if err := configs.NewDelete().
							Where(gorp.MatchKeys[string, taskConfig](c.Key)).
							Exec(ctx, tx); err != nil {
							return err
						}
						return moved.NewCreate().
							Entry(&movedTaskConfig{taskConfig: c}).
							Exec(ctx, tx)
					},
					needed: need{access.ActionUpdate, task.ResourceID(t.Key)},
				}
			},
		),
		Entry("update on the task of a deleted task config record", func(
			ctx context.Context,
		) rule {
			configs := openConfigs[taskConfig](ctx)
			t := createTask(ctx)
			k := t.Key.String()
			Expect(configs.NewCreate().
				Entry(&taskConfig{Key: k}).
				Exec(ctx, db)).To(Succeed())
			return rule{
				write: func(ctx context.Context, tx gorp.Tx) error {
					return configs.NewDelete().
						Where(gorp.MatchKeys[string, taskConfig](k)).
						Exec(ctx, tx)
				},
				needed: need{access.ActionUpdate, task.ResourceID(t.Key)},
			}
		}),
	)

	It(
		"Should not require access on the target of a use link removed",
		func(ctx SpecContext) {
			u := newUser(ctx)
			l := createLabel(ctx)
			r := createRange(ctx, nil, l.Key)
			grant(ctx, u, access.ActionUpdate, r.ResourceID())
			Expect(write(ctx, u, func(ctx context.Context, tx gorp.Tx) error {
				return svc.Ranger.NewWriter(tx).
					RemoveLabels(ctx, r.Key, []label.Key{l.Key})
			})).To(Succeed())
		},
	)

	DescribeTable(
		"Should not require access on the target of a link when the write creates or "+
			"deletes the target",
		func(ctx SpecContext, setup func(ctx context.Context) (apply, []need)) {
			u := newUser(ctx)
			f, granted := setup(ctx)
			for _, g := range granted {
				grant(ctx, u, g.action, g.object)
			}
			Expect(write(ctx, u, f)).To(Succeed())
		},
		Entry("a place target created", func(context.Context) (apply, []need) {
			parent := newRange(nil)
			child := newRange(&parent.Key)
			return func(ctx context.Context, tx gorp.Tx) error {
				return svc.Ranger.NewWriter(tx).
					CreateMany(ctx, &[]ranger.Range{parent, child})
			}, []need{
				{access.ActionCreate, parent.ResourceID()},
				{access.ActionCreate, child.ResourceID()},
			}
		}),
		Entry("a place target deleted", func(ctx context.Context) (apply, []need) {
			parent := createRange(ctx, nil)
			child := createRange(ctx, &parent.Key)
			return func(ctx context.Context, tx gorp.Tx) error {
				return svc.Ranger.NewWriter(tx).Delete(ctx, parent.Key)
			}, []need{
				{access.ActionDelete, parent.ResourceID()},
				{access.ActionDelete, child.ResourceID()},
			}
		}),
		Entry("a use target created", func(context.Context) (apply, []need) {
			l := label.Label{Key: uuid.New(), Name: "new"}
			r := newRange(nil, l.Key)
			return func(ctx context.Context, tx gorp.Tx) error {
				if err := svc.Label.NewWriter(tx).Create(ctx, &l); err != nil {
					return err
				}
				return svc.Ranger.NewWriter(tx).Create(ctx, &r)
			}, []need{
				{access.ActionCreate, l.ResourceID()},
				{access.ActionCreate, r.ResourceID()},
			}
		}),
	)

	DescribeTable(
		"Should not require access on the owner of a record when the write creates or "+
			"deletes the owner",
		func(ctx SpecContext, setup func(ctx context.Context) (apply, need)) {
			u := newUser(ctx)
			f, granted := setup(ctx)
			grant(ctx, u, granted.action, granted.object)
			Expect(write(ctx, u, f)).To(Succeed())
		},
		Entry("an owner created", func(context.Context) (apply, need) {
			r := newRange(nil)
			return func(ctx context.Context, tx gorp.Tx) error {
				if err := svc.Ranger.NewWriter(tx).Create(ctx, &r); err != nil {
					return err
				}
				return setPair(ctx, tx, r.Key, "new")
			}, need{access.ActionCreate, r.ResourceID()}
		}),
		Entry("an owner deleted", func(ctx context.Context) (apply, need) {
			r := createRange(ctx, nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return setPair(ctx, tx, r.Key, "stored")
			})).To(Succeed())
			return func(ctx context.Context, tx gorp.Tx) error {
				return svc.Ranger.NewWriter(tx).Delete(ctx, r.Key)
			}, need{access.ActionDelete, r.ResourceID()}
		}),
	)

	It("Should let a user write its own API keys", func(ctx SpecContext) {
		u := newUser(ctx)
		var k apikey.APIKey
		Expect(write(ctx, u, func(ctx context.Context, tx gorp.Tx) (err error) {
			k, _, err = svc.APIKey.NewWriter(tx).Create(ctx, u.ResourceID(), "own")
			return err
		})).To(Succeed())
		Expect(write(ctx, u, func(ctx context.Context, tx gorp.Tx) error {
			return svc.APIKey.NewWriter(tx).Delete(ctx, k.Key)
		})).To(Succeed())
	})

	It(
		"Should require update on the user that loses an API key to the subject",
		func(ctx SpecContext) {
			u, other := newUser(ctx), newUser(ctx)
			var k apikey.APIKey
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				k, _, err = svc.APIKey.NewWriter(tx).
					Create(ctx, other.ResourceID(), "key")
				return err
			})).To(Succeed())
			Expect(write(ctx, u, func(ctx context.Context, tx gorp.Tx) error {
				return gorp.NewUpdate[apikey.Key, apikey.APIKey]().
					Where(gorp.MatchKeys[apikey.Key, apikey.APIKey](k.Key)).
					Change(func(_ gorp.Context, k apikey.APIKey) apikey.APIKey {
						k.Subject = u.ResourceID()
						return k
					}).
					Exec(ctx, tx)
			})).To(denied(u, access.ActionUpdate, other.ResourceID()))
		},
	)

	It("Should roll back every write when access is denied", func(ctx SpecContext) {
		u := newUser(ctx)
		allowed := label.Label{Key: uuid.New(), Name: "allowed"}
		refused := label.Label{Key: uuid.New(), Name: "refused"}
		grant(ctx, u, access.ActionCreate, allowed.ResourceID())
		Expect(write(ctx, u, func(ctx context.Context, tx gorp.Tx) error {
			return svc.Label.NewWriter(tx).
				CreateMany(ctx, &[]label.Label{allowed, refused})
		})).To(denied(u, access.ActionCreate, refused.ResourceID()))
		Expect(labelExists(ctx, allowed.Key)).To(BeFalse())
		Expect(labelExists(ctx, refused.Key)).To(BeFalse())
	})

	It(
		"Should return an error for a write to a table with no resource type",
		func(ctx SpecContext) {
			u := newUser(ctx)
			Expect(write(ctx, u, func(ctx context.Context, tx gorp.Tx) error {
				return gorp.NewCreate[string, unmapped]().
					Entry(&unmapped{Key: "k"}).
					Exec(ctx, tx)
			})).To(MatchError(
				"[bind] - table unmapped has no resource type, and k is placed in 0 " +
					"records",
			))
			Expect(gorp.NewRetrieve[string, unmapped]().
				Where(gorp.MatchKeys[string, unmapped]("k")).
				Entry(&unmapped{}).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		},
	)

	It(
		"Should return an error for a record with no resource type that is placed in "+
			"two records",
		func(ctx SpecContext) {
			u := newUser(ctx)
			first, second := createRange(ctx, nil), createRange(ctx, nil)
			rng := string(resource.TypeRange)
			table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, placedTwice]{
				DB: db,
				References: []gorp.Reference[string, placedTwice]{
					gorp.NewTargetReference(
						gorp.TargetReferenceConfig[string, placedTwice]{
							Name:     "ranges",
							Tables:   []string{rng},
							OnDelete: gorp.Cascade,
							Kind:     gorp.Place,
							Targets: func(p *placedTwice) []gorp.Target {
								return []gorp.Target{
									gorp.NewTarget(rng, p.First),
									gorp.NewTarget(rng, p.Second),
								}
							},
						},
					),
				},
			}))
			Expect(write(ctx, u, func(ctx context.Context, tx gorp.Tx) error {
				return table.NewCreate().Entry(&placedTwice{
					Key:    "k",
					First:  first.Key.String(),
					Second: second.Key.String(),
				}).Exec(ctx, tx)
			})).To(MatchError(
				"[bind] - table placedTwice has no resource type, and k is placed in " +
					"2 records",
			))
		},
	)

	DescribeTable(
		"Should ignore access that the write itself grants",
		func(ctx SpecContext, stage func(context.Context, gorp.Tx, user.User) error) {
			u := newUser(ctx)
			f := func(ctx context.Context, tx gorp.Tx) error {
				if err := stage(ctx, tx, u); err != nil {
					return err
				}
				l := label.Label{Name: "staged"}
				return svc.Label.NewWriter(tx).Create(ctx, &l)
			}
			Expect(write(ctx, u, f)).To(MatchError(access.ErrDenied))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return stage(ctx, tx, u)
			})).To(Succeed())
			Expect(write(ctx, u, f)).To(Succeed())
		},
		Entry(
			"a policy",
			func(ctx context.Context, tx gorp.Tx, u user.User) error {
				p := policy.Policy{
					Name: "staged",
					Role: u.Role,
					Objects: []resource.ID{
						{Type: resource.TypeLabel},
						{Type: resource.TypePolicy},
						{Type: resource.TypeRole},
					},
					Actions: access.AllActions,
				}
				return svc.RBAC.Policy.NewWriter(tx, false).Create(ctx, &p)
			},
		),
		Entry(
			"a role",
			func(ctx context.Context, tx gorp.Tx, u user.User) error {
				return svc.User.NewWriter(tx).
					SetRole(ctx, svc.RBAC.Builtin.OwnerKey, u.Key)
			},
		),
	)

	It(
		"Should require retrieve on a returned record that the write did not change",
		func(ctx SpecContext) {
			u := newUser(ctx)
			l := createLabel(ctx)
			unchanged := func(context.Context, gorp.Tx) error { return nil }
			Expect(writeReturning(ctx, u, unchanged, l.ResourceID())).
				To(denied(u, access.ActionRetrieve, l.ResourceID()))
			grant(ctx, u, access.ActionRetrieve, l.ResourceID())
			Expect(writeReturning(ctx, u, unchanged, l.ResourceID())).To(Succeed())
		},
	)

	It(
		"Should require only create on a returned record that the write created",
		func(ctx SpecContext) {
			u := newUser(ctx)
			l := label.Label{Key: uuid.New(), Name: "returned"}
			grant(ctx, u, access.ActionCreate, l.ResourceID())
			Expect(writeReturning(ctx, u, func(ctx context.Context, tx gorp.Tx) error {
				return svc.Label.NewWriter(tx).Create(ctx, &l)
			}, l.ResourceID())).To(Succeed())
		},
	)

	It("Should return the response of apply", func(ctx SpecContext) {
		u := newUser(ctx)
		Expect(bind.Write(
			binder,
			func(context.Context, gorp.Tx, string) (string, error) {
				return "response", nil
			},
		)(AuthedCtx(ctx, u), "request")).To(Equal("response"))
	})

	It(
		"Should return a zero response and roll back when apply fails",
		func(ctx SpecContext) {
			u := newUser(ctx)
			l := label.Label{Key: uuid.New(), Name: "failed"}
			grant(ctx, u, access.ActionCreate, l.ResourceID())
			failed := errors.New("failed")
			Expect(bind.Write(
				binder,
				func(ctx context.Context, tx gorp.Tx, _ struct{}) (string, error) {
					if err := svc.Label.NewWriter(tx).Create(ctx, &l); err != nil {
						return "", err
					}
					return "response", failed
				},
			)(AuthedCtx(ctx, u), struct{}{})).Error().To(MatchError(failed))
			Expect(labelExists(ctx, l.Key)).To(BeFalse())
		},
	)
})

var _ = Describe("WritePrepared", func() {
	It("Should pass what prepare returns to apply", func(ctx SpecContext) {
		u := newUser(ctx)
		Expect(bind.WritePrepared(
			binder,
			func(_ context.Context, req string) (string, error) {
				return req + " prepared", nil
			},
			func(_ context.Context, _ gorp.Tx, _, prepared string) (string, error) {
				return prepared, nil
			},
		)(AuthedCtx(ctx, u), "request")).To(Equal("request prepared"))
	})
	It("Should return the error of prepare without running apply", func(
		ctx SpecContext,
	) {
		u := newUser(ctx)
		failed := errors.New("failed")
		applied := false
		Expect(bind.WritePrepared(
			binder,
			func(context.Context, string) (string, error) { return "", failed },
			func(context.Context, gorp.Tx, string, string) (string, error) {
				applied = true
				return "response", nil
			},
		)(AuthedCtx(ctx, u), "request")).Error().To(MatchError(failed))
		Expect(applied).To(BeFalse())
	})
})
