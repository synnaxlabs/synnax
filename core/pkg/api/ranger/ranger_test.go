// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"slices"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apiranger "github.com/synnaxlabs/synnax/pkg/api/ranger"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Retrieve", func() {
	It("Should omit a missing key instead of failing", func(ctx SpecContext) {
		u := createUser(ctx)
		r := apiranger.Range{
			Key:       uuid.New(),
			Name:      "present",
			TimeRange: telem.TimeRange{Start: 1, End: 2},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		missing := uuid.New()
		grant(
			ctx,
			access.ActionRetrieve,
			u.ResourceID(),
			r.ResourceID(),
			ranger.ResourceID(missing),
		)
		res := MustSucceed(retrieve(
			AuthedCtx(ctx, u),
			apiranger.RetrieveRequest{Where: &ranger.Where{
				Key: &filter.Set[apiranger.Key]{In: []apiranger.Key{r.Key, missing}},
			}},
		))
		Expect(res.Ranges).To(HaveExactElements(HaveField("Key", r.Key)))
	})

	DescribeTable("Should keep a child range only when its parent passes", func(
		ctx SpecContext,
		parentGranted bool,
		expected int,
	) {
		u := createUser(ctx)
		parent := createRange(ctx)
		child := ranger.Range{
			Name:      "child",
			TimeRange: telem.NewRangeSeconds(1, 5),
			Parent:    &parent.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).Create(ctx, &child)
		})).To(Succeed())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), child.ResourceID())
		if parentGranted {
			grant(ctx, access.ActionRetrieve, u.ResourceID(), parent.ResourceID())
		}
		res := MustSucceed(retrieve(
			AuthedCtx(ctx, u),
			apiranger.RetrieveRequest{Where: &ranger.Where{
				Parents: []ranger.Key{parent.Key},
			}},
		))
		Expect(res.Ranges).To(HaveLen(expected))
	},
		Entry("with the parent granted", true, 1),
		Entry("with the parent denied", false, 0),
	)

	DescribeTable("Should count only the children that the subject can retrieve", func(
		ctx SpecContext,
		childGranted bool,
	) {
		u := createUser(ctx)
		parent := createRange(ctx)
		child := ranger.Range{
			Name:      "child",
			TimeRange: telem.NewRangeSeconds(1, 5),
			Parent:    &parent.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).Create(ctx, &child)
		})).To(Succeed())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), parent.ResourceID())
		if childGranted {
			grant(ctx, access.ActionRetrieve, u.ResourceID(), child.ResourceID())
		}
		res := MustSucceed(retrieve(
			AuthedCtx(ctx, u),
			apiranger.RetrieveRequest{Where: &ranger.Where{
				Key: &filter.Set[ranger.Key]{Eq: &parent.Key},
			}},
		))
		Expect(res.Ranges).To(HaveExactElements(
			HaveField("HasChildren", childGranted),
		))
	},
		Entry("with the child granted", true),
		Entry("with the child denied", false),
	)

	It("Should deny a named child range whose parent the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		parent := createRange(ctx)
		child := ranger.Range{
			Name:      "named child",
			TimeRange: telem.NewRangeSeconds(1, 5),
			Parent:    &parent.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).Create(ctx, &child)
		})).To(Succeed())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), child.ResourceID())
		Expect(retrieve(
			AuthedCtx(ctx, u),
			apiranger.RetrieveRequest{Where: &ranger.Where{
				Key: &filter.Set[ranger.Key]{Eq: &child.Key},
			}},
		)).Error().To(MatchError(access.Denied(
			u.ResourceID(),
			access.ActionRetrieve,
			parent.ResourceID(),
		).Error()))
	})

	It("Should return no range for a zero limit", func(ctx SpecContext) {
		u := createUser(ctx)
		r := apiranger.Range{
			Key:       uuid.New(),
			Name:      "limited",
			TimeRange: telem.TimeRange{Start: 1, End: 2},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), r.ResourceID())
		res := MustSucceed(retrieve(
			AuthedCtx(ctx, u),
			apiranger.RetrieveRequest{
				Where: &ranger.Where{Key: &filter.Set[apiranger.Key]{Eq: &r.Key}},
				Limit: new(uint32(0)),
			},
		))
		Expect(res.Ranges).To(BeEmpty())
	})

	It("Should return a path error under where for an invalid where", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		Expect(retrieve(AuthedCtx(ctx, u), apiranger.RetrieveRequest{
			Where: &ranger.Where{Name: &filter.String{Regex: new("(a")}},
		})).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError("where.name.regex: error parsing regexp: missing closing ): "+
				"`(a`: validation error"),
		))
	})
})

var _ = Describe("RetrieveRequest", func() {
	DescribeTable(
		"Should reject a member that the request does not declare",
		func(body string) {
			var req apiranger.RetrieveRequest
			Expect(json.Unmarshal([]byte(body), &req)).To(SatisfyAll(
				MatchError(json.ErrUnknownName),
				MatchError(ContainSubstring(`unknown object member name "keys"`)),
			))
		},
		Entry("at the top", `{"keys":["00000000-0000-4000-8000-00000000000a"]}`),
		Entry("in a nested where", `{"where":{"or":[{"keys":[]}]}}`),
	)

	DescribeTable(
		"Should reject a MessagePack member that the request does not declare",
		func(body map[string]any) {
			data := MustSucceed(msgpack.Marshal(body))
			var req apiranger.RetrieveRequest
			Expect(msgpack.Unmarshal(data, &req)).
				To(MatchError(`msgpack: unknown field "keys"`))
		},
		Entry("at the top", map[string]any{"keys": []string{"a"}}),
		Entry("in a nested where", map[string]any{
			"where": map[string]any{"or": []any{map[string]any{"keys": []any{}}}},
		}),
	)

	It("Should decode a MessagePack request that it declares", func() {
		data := MustSucceed(msgpack.Marshal(map[string]any{
			"where": map[string]any{"name": map[string]any{"eq": "a"}},
			"limit": 5,
		}))
		var req apiranger.RetrieveRequest
		Expect(msgpack.Unmarshal(data, &req)).To(Succeed())
		Expect(req.Where.Name.Eq).To(HaveValue(Equal("a")))
		Expect(req.Limit).To(HaveValue(BeEquivalentTo(5)))
	})
})

func createRange(ctx SpecContext) ranger.Range {
	GinkgoHelper()
	r := ranger.Range{Name: "Hot fire", TimeRange: telem.NewRangeSeconds(1, 5)}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Ranger.NewWriter(tx).Create(ctx, &r)
	})).To(Succeed())
	return r
}

func rangeExists(ctx SpecContext, key ranger.Key) bool {
	GinkgoHelper()
	return MustSucceed(svc.Ranger.NewRetrieve().
		Where(ranger.MatchKeys(key)).
		Exists(ctx, db))
}

func retrieveRange(ctx SpecContext, key ranger.Key) ranger.Range {
	GinkgoHelper()
	var r ranger.Range
	Expect(svc.Ranger.NewRetrieve().
		Where(ranger.MatchKeys(key)).
		Entry(&r).
		Exec(ctx, db)).To(Succeed())
	return r
}

var _ = Describe("Dispatch", func() {
	rename := []ranger.Action{ranger.NewSetAction(ranger.SetPayload{
		Path:  []string{"name"},
		Value: "Cold flow",
	})}
	It("Should apply the actions when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), r.ResourceID())
		MustSucceed(dispatch(
			AuthedCtx(ctx, u),
			apiranger.DispatchRequest{Key: r.Key, DispatchKey: "dk", Actions: rename},
		))
		Expect(retrieveRange(ctx, r.Key).Name).To(Equal("Cold flow"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		Expect(dispatch(
			AuthedCtx(ctx, u),
			apiranger.DispatchRequest{Key: r.Key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveRange(ctx, r.Key).Name).To(Equal("Hot fire"))
	})
	moveUnder := func(parent ranger.Key) []ranger.Action {
		return []ranger.Action{ranger.NewSetAction(ranger.SetPayload{
			Path:  []string{"parent"},
			Value: parent.String(),
		})}
	}
	It(
		"Should refuse a move under a parent the subject cannot update",
		func(ctx SpecContext) {
			u := createUser(ctx)
			child := createRange(ctx)
			parent := createRange(ctx)
			grant(ctx, access.ActionUpdate, u.ResourceID(), child.ResourceID())
			Expect(dispatch(
				AuthedCtx(ctx, u),
				apiranger.DispatchRequest{
					Key:     child.Key,
					Actions: moveUnder(parent.Key),
				},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveRange(ctx, child.Key).Parent).To(BeNil())
		},
	)
	It(
		"Should refuse a move out of a parent the subject cannot update",
		func(ctx SpecContext) {
			u := createUser(ctx)
			oldParent := createRange(ctx)
			newParent := createRange(ctx)
			child := ranger.Range{
				Name:      "Child",
				TimeRange: telem.NewRangeSeconds(1, 5),
				Parent:    &oldParent.Key,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Ranger.NewWriter(tx).Create(ctx, &child)
			})).To(Succeed())
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				child.ResourceID(),
				newParent.ResourceID(),
			)
			Expect(dispatch(
				AuthedCtx(ctx, u),
				apiranger.DispatchRequest{
					Key:     child.Key,
					Actions: moveUnder(newParent.Key),
				},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveRange(ctx, child.Key).Parent).
				To(HaveValue(Equal(oldParent.Key)))
		},
	)
	It("Should move a range when the subject can update both parents", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		oldParent := createRange(ctx)
		newParent := createRange(ctx)
		child := ranger.Range{
			Name:      "Child",
			TimeRange: telem.NewRangeSeconds(1, 5),
			Parent:    &oldParent.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).Create(ctx, &child)
		})).To(Succeed())
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			child.ResourceID(),
			oldParent.ResourceID(),
			newParent.ResourceID(),
		)
		MustSucceed(dispatch(
			AuthedCtx(ctx, u),
			apiranger.DispatchRequest{
				Key:     child.Key,
				Actions: moveUnder(newParent.Key),
			},
		))
		Expect(retrieveRange(ctx, child.Key).Parent).To(HaveValue(Equal(newParent.Key)))
	})
	It("Should return the dispatch error", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.ResourceID(), ranger.ResourceID(key))
		Expect(dispatch(
			AuthedCtx(ctx, u),
			apiranger.DispatchRequest{Key: key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Create", func() {
	It(
		"Should stamp the dispatch key onto the create broadcast",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{
				Type: resource.TypeRange,
			})
			rec := &Recorder[ranger.Key, ranger.Action]{}
			DeferCleanup(svc.Ranger.OnAction(rec.Record))
			res := MustSucceed(
				create(AuthedCtx(ctx, u), apiranger.CreateRequest{
					Ranges: []ranger.Range{{
						Name:      "Created",
						TimeRange: telem.NewRangeSeconds(1, 5),
					}},
					DispatchKey: "dk-create",
				}),
			)
			Expect(res.Ranges).To(HaveLen(1))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		},
	)

	It("Should refuse a create without create on the range", func(ctx SpecContext) {
		u := createUser(ctx)
		r := ranger.Range{
			Key:       uuid.New(),
			Name:      "Denied",
			TimeRange: telem.NewRangeSeconds(1, 5),
		}
		Expect(create(AuthedCtx(ctx, u), apiranger.CreateRequest{
			Ranges: []ranger.Range{r},
		})).Error().To(denied(u, access.ActionCreate, r.ResourceID()))
		Expect(rangeExists(ctx, r.Key)).To(BeFalse())
	})

	It("Should refuse a range whose key exists", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{
			Type: resource.TypeRange,
		})
		stored := createRange(ctx)
		overwrite := stored
		overwrite.Name = "Overwritten"
		Expect(create(AuthedCtx(ctx, u), apiranger.CreateRequest{
			Ranges: []ranger.Range{overwrite},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("range %s already exists", stored.Key)),
		))
		Expect(retrieveRange(ctx, stored.Key).Name).To(Equal("Hot fire"))
	})

	Context("Under a parent", func() {
		var (
			u      user.User
			parent ranger.Range
			child  ranger.Range
		)
		BeforeEach(func(ctx SpecContext) {
			u = createUser(ctx)
			parent = createRange(ctx)
			child = ranger.Range{
				Key:       uuid.New(),
				Name:      "Child",
				TimeRange: telem.NewRangeSeconds(1, 5),
				Parent:    &parent.Key,
			}
			grant(ctx, access.ActionCreate, u.ResourceID(), child.ResourceID())
		})
		createChild := func(ctx SpecContext) (apiranger.CreateResponse, error) {
			return create(AuthedCtx(ctx, u), apiranger.CreateRequest{
				Ranges: []ranger.Range{child},
			})
		}
		It(
			"Should refuse the create without update on the parent",
			func(ctx SpecContext) {
				grant(ctx, access.ActionCreate, u.ResourceID(), parent.ResourceID())
				Expect(createChild(ctx)).Error().
					To(denied(u, access.ActionUpdate, parent.ResourceID()))
				Expect(rangeExists(ctx, child.Key)).To(BeFalse())
			},
		)
		It("Should create the range with update on the parent", func(ctx SpecContext) {
			grant(ctx, access.ActionUpdate, u.ResourceID(), parent.ResourceID())
			MustSucceed(createChild(ctx))
			Expect(retrieveRange(ctx, child.Key).Parent).
				To(HaveValue(Equal(parent.Key)))
		})
	})

	Context("With a label", func() {
		var (
			u user.User
			l label.Label
			r ranger.Range
		)
		BeforeEach(func(ctx SpecContext) {
			u = createUser(ctx)
			l = label.Label{Name: "Labeled"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Label.NewWriter(tx).Create(ctx, &l)
			})).To(Succeed())
			r = ranger.Range{
				Key:       uuid.New(),
				Name:      "Labeled",
				TimeRange: telem.NewRangeSeconds(1, 5),
				Labels:    []label.Key{l.Key},
			}
			grant(ctx, access.ActionCreate, u.ResourceID(), r.ResourceID())
		})
		createLabeled := func(ctx SpecContext) (apiranger.CreateResponse, error) {
			return create(AuthedCtx(ctx, u), apiranger.CreateRequest{
				Ranges: []ranger.Range{r},
			})
		}
		It(
			"Should refuse the create without retrieve on the label",
			func(ctx SpecContext) {
				grant(ctx, access.ActionCreate, u.ResourceID(), l.ResourceID())
				Expect(createLabeled(ctx)).Error().
					To(denied(u, access.ActionRetrieve, l.ResourceID()))
				Expect(rangeExists(ctx, r.Key)).To(BeFalse())
			},
		)
		It("Should create the range with retrieve on the label", func(ctx SpecContext) {
			grant(ctx, access.ActionRetrieve, u.ResourceID(), l.ResourceID())
			MustSucceed(createLabeled(ctx))
			Expect(retrieveRange(ctx, r.Key).Labels).To(Equal([]label.Key{l.Key}))
		})
	})
})

var _ = Describe("Delete", func() {
	var (
		u      user.User
		parent ranger.Range
		child  ranger.Range
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		parent = createRange(ctx)
		child = ranger.Range{
			Name:      "Child",
			TimeRange: telem.NewRangeSeconds(1, 5),
			Parent:    &parent.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).Create(ctx, &child)
		})).To(Succeed())
	})
	deleteChild := func(ctx SpecContext) (struct{}, error) {
		return del(AuthedCtx(ctx, u), apiranger.DeleteRequest{
			Keys: []ranger.Key{child.Key},
		})
	}
	childID := func() resource.ID { return child.ResourceID() }
	parentID := func() resource.ID { return parent.ResourceID() }
	DescribeTable(
		"Should refuse the delete and keep the range",
		func(
			ctx SpecContext,
			granted access.Action,
			grantedOn func() resource.ID,
			refused access.Action,
			refusedOn func() resource.ID,
		) {
			grant(ctx, granted, u.ResourceID(), grantedOn())
			Expect(deleteChild(ctx)).Error().To(denied(u, refused, refusedOn()))
			Expect(rangeExists(ctx, child.Key)).To(BeTrue())
		},
		Entry(
			"without delete on the range",
			access.ActionUpdate,
			parentID,
			access.ActionDelete,
			childID,
		),
		Entry(
			"without update on its parent",
			access.ActionDelete,
			childID,
			access.ActionUpdate,
			parentID,
		),
	)
	It(
		"Should delete the range with delete on it and update on its parent",
		func(ctx SpecContext) {
			grant(ctx, access.ActionDelete, u.ResourceID(), childID())
			grant(ctx, access.ActionUpdate, u.ResourceID(), parentID())
			MustSucceed(deleteChild(ctx))
			Expect(rangeExists(ctx, child.Key)).To(BeFalse())
		},
	)
	It(
		"Should not need update on the range to delete its key-value pairs with it",
		func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.KV.NewWriter(tx).SetMany(ctx, []kv.Pair{
					{Range: child.Key, Key: "k", Value: "v"},
				})
			})).To(Succeed())
			grant(ctx, access.ActionDelete, u.ResourceID(), childID())
			grant(ctx, access.ActionUpdate, u.ResourceID(), parentID())
			MustSucceed(deleteChild(ctx))
			Expect(rangeExists(ctx, child.Key)).To(BeFalse())
		},
	)
})

// held is a record in a range that a range delete cascades into.
type held struct {
	id     resource.ID
	exists func(ctx SpecContext) bool
}

var _ = Describe("Delete cascades", func() {
	var (
		u user.User
		r ranger.Range
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		r = createRange(ctx)
	})
	deleteRange := func(ctx SpecContext) (struct{}, error) {
		return del(
			AuthedCtx(ctx, u),
			apiranger.DeleteRequest{Keys: []ranger.Key{r.Key}},
		)
	}
	createChild := func(ctx SpecContext) held {
		GinkgoHelper()
		child := ranger.Range{
			Name:      "Child",
			TimeRange: telem.NewRangeSeconds(1, 5),
			Parent:    &r.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).Create(ctx, &child)
		})).To(Succeed())
		return held{
			id: child.ResourceID(),
			exists: func(ctx SpecContext) bool {
				GinkgoHelper()
				return rangeExists(ctx, child.Key)
			},
		}
	}
	createAlias := func(ctx SpecContext) held {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     "aliased_" + strings.ReplaceAll(uuid.New().String(), "-", "_"),
			DataType: telem.Float64T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Alias.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "alias")
		})).To(Succeed())
		return held{
			id: alias.ResourceID(r.Key, ch.Key()),
			exists: func(ctx SpecContext) bool {
				GinkgoHelper()
				_, err := svc.Alias.NewReader(db).Resolve(ctx, r.Key, "alias")
				Expect(errors.Skip(err, query.ErrNotFound)).To(Succeed())
				return err == nil
			},
		}
	}
	createLinePlot := func(ctx SpecContext) held {
		GinkgoHelper()
		lp := defaults.Fill(lineplot.LinePlot{Name: "Plot", Range: &r.Key})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.LinePlot.NewWriter(tx).Create(ctx, &lp)
		})).To(Succeed())
		return held{
			id: lineplot.ResourceID(lp.Key),
			exists: func(ctx SpecContext) bool {
				GinkgoHelper()
				return MustSucceed(svc.LinePlot.NewRetrieve().
					Where(lineplot.MatchKeys(lp.Key)).
					Exists(ctx, db))
			},
		}
	}
	createSnapshot := func(ctx SpecContext) held {
		GinkgoHelper()
		t := task.Task{
			Type:     pagerduty.AlertTaskType,
			Rack:     svc.Rack.EmbeddedKey,
			Name:     "Snapshot",
			Snapshot: true,
			Range:    &r.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Task.NewWriter(tx).Create(ctx, &t)
		})).To(Succeed())
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			t.ResourceID(),
			svc.Rack.EmbeddedKey.ResourceID(),
		)
		grant(ctx, access.ActionDelete, u.ResourceID(), resource.ID{
			Type: resource.TypeStatus,
		})
		return held{
			id: t.ResourceID(),
			exists: func(ctx SpecContext) bool {
				GinkgoHelper()
				return MustSucceed(svc.Task.NewRetrieve().
					Where(task.MatchKeys(t.Key)).
					Exists(ctx, db))
			},
		}
	}
	DescribeTable(
		"Should need delete on each record that the range holds",
		func(ctx SpecContext, create func(ctx SpecContext) held) {
			h := create(ctx)
			grant(ctx, access.ActionDelete, u.ResourceID(), r.ResourceID())
			grant(ctx, access.ActionUpdate, u.ResourceID(), r.ResourceID())
			Expect(deleteRange(ctx)).Error().
				To(denied(u, access.ActionDelete, h.id))
			Expect(rangeExists(ctx, r.Key)).To(BeTrue())
			Expect(h.exists(ctx)).To(BeTrue())
			grant(ctx, access.ActionDelete, u.ResourceID(), h.id)
			MustSucceed(deleteRange(ctx))
			Expect(rangeExists(ctx, r.Key)).To(BeFalse())
			Expect(h.exists(ctx)).To(BeFalse())
		},
		Entry("a nested range", createChild),
		Entry("an alias", createAlias),
		Entry("a line plot", createLinePlot),
		Entry("a snapshot task", createSnapshot),
	)
})

var _ = Describe("Retrieve", func() {
	authed := func(ctx SpecContext) context.Context {
		GinkgoHelper()
		u := createUser(ctx)
		grant(
			ctx,
			access.ActionRetrieve,
			u.ResourceID(),
			resource.ID{Type: resource.TypeRange},
			resource.ID{Type: resource.TypeLabel},
		)
		return AuthedCtx(ctx, u)
	}
	keysOf := func(ranges []apiranger.Range) []ranger.Key {
		return lo.Map(
			ranges,
			func(r apiranger.Range, _ int) ranger.Key { return r.Key },
		)
	}

	It("Should retrieve the children of the given parents", func(ctx SpecContext) {
		parent := createRange(ctx)
		child := ranger.Range{
			Name:      "Burn",
			TimeRange: telem.NewRangeSeconds(1, 2),
			Parent:    &parent.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).Create(ctx, &child)
		})).To(Succeed())
		createRange(ctx)
		res := MustSucceed(retrieve(authed(ctx), apiranger.RetrieveRequest{
			Where: &ranger.Where{Parents: []ranger.Key{parent.Key}},
		}))
		Expect(keysOf(res.Ranges)).To(Equal([]ranger.Key{child.Key}))
		Expect(res.Ranges[0].Parent).To(HaveValue(Equal(parent.Key)))
	})

	It("Should retrieve the ranges that hold any of the labels", func(ctx SpecContext) {
		a, b := label.Label{Name: "A"}, label.Label{Name: "B"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Label.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Label.NewWriter(tx).Create(ctx, &b)
		})).To(Succeed())
		withA, withB := createRange(ctx), createRange(ctx)
		createRange(ctx)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).
				AddLabels(ctx, withA.Key, []label.Key{a.Key})
		})).
			To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).
				AddLabels(ctx, withB.Key, []label.Key{b.Key})
		})).
			To(Succeed())
		res := MustSucceed(retrieve(authed(ctx), apiranger.RetrieveRequest{
			Where: &ranger.Where{Labels: []label.Key{a.Key, b.Key}},
		}))
		Expect(keysOf(res.Ranges)).To(ConsistOf(withA.Key, withB.Key))
	})

	Describe("Order", func() {
		var (
			parent ranger.Range
			// byName holds the children in name order, with the tie in key order.
			byName []ranger.Key
			// byStart holds the children from the latest start to the earliest.
			byStart []ranger.Key
			// byEnd holds the children from the earliest end to the latest.
			byEnd []ranger.Key
		)
		BeforeEach(func(ctx SpecContext) {
			parent = createRange(ctx)
			child := func(name string, start, end telem.TimeStamp) ranger.Key {
				GinkgoHelper()
				r := ranger.Range{
					Name:      name,
					TimeRange: telem.TimeRange{Start: start, End: end},
					Parent:    &parent.Key,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Ranger.NewWriter(tx).Create(ctx, &r)
				})).To(Succeed())
				return r.Key
			}
			beta := child("beta", 30, 35)
			upper := child("Alpha", 10, 50)
			gamma := child("Gamma", 20, 25)
			lower := child("alpha", 40, 41)
			tie := []ranger.Key{upper, lower}
			// Gorp orders equal names by the trailing bytes of the key.
			slices.SortFunc(tie, func(a, b ranger.Key) int {
				slices.Reverse(a[:])
				slices.Reverse(b[:])
				return bytes.Compare(a[:], b[:])
			})
			byName = append(tie, beta, gamma)
			byStart = []ranger.Key{lower, beta, gamma, upper}
			byEnd = []ranger.Key{gamma, beta, lower, upper}
		})
		page := func(
			ctx context.Context,
			order apiranger.Order,
			limit uint32,
		) []ranger.Key {
			GinkgoHelper()
			var (
				keys  []ranger.Key
				after *apiranger.Cursor
			)
			for {
				res := MustSucceed(retrieve(ctx, apiranger.RetrieveRequest{
					Where: &ranger.Where{Parents: []ranger.Key{parent.Key}},
					Order: &order,
					After: after,
					Limit: &limit,
				}))
				if len(res.Ranges) == 0 {
					return keys
				}
				keys = append(keys, keysOf(res.Ranges)...)
				last := res.Ranges[len(res.Ranges)-1]
				after = &apiranger.Cursor{
					Key:            last.Key,
					Name:           &last.Name,
					TimeRangeStart: &last.TimeRange.Start,
					TimeRangeEnd:   &last.TimeRange.End,
				}
			}
		}

		It("Should page ranges by name, ignoring case", func(ctx SpecContext) {
			order := apiranger.Order{Field: apiranger.OrderFieldName}
			Expect(page(authed(ctx), order, 3)).To(Equal(byName))
		})

		It("Should page ranges by name in descending order", func(ctx SpecContext) {
			order := apiranger.Order{Field: apiranger.OrderFieldName, Descending: true}
			desc := slices.Clone(byName)
			slices.Reverse(desc)
			Expect(page(authed(ctx), order, 1)).To(Equal(desc))
		})

		It("Should page ranges by start time in descending order", func(
			ctx SpecContext,
		) {
			order := apiranger.Order{
				Field:      apiranger.OrderFieldTimeRangeStart,
				Descending: true,
			}
			Expect(page(authed(ctx), order, 2)).To(Equal(byStart))
		})

		It("Should page ranges by end time", func(ctx SpecContext) {
			order := apiranger.Order{Field: apiranger.OrderFieldTimeRangeEnd}
			Expect(page(authed(ctx), order, 3)).To(Equal(byEnd))
		})

		It("Should page ranges by key with no order", func(ctx SpecContext) {
			authedCtx := authed(ctx)
			all := MustSucceed(retrieve(authedCtx, apiranger.RetrieveRequest{
				Where: &ranger.Where{Parents: []ranger.Key{parent.Key}},
				Limit: new(uint32(2)),
			}))
			Expect(all.Ranges).To(HaveLen(2))
			rest := MustSucceed(retrieve(authedCtx, apiranger.RetrieveRequest{
				Where: &ranger.Where{Parents: []ranger.Key{parent.Key}},
				After: &apiranger.Cursor{Key: all.Ranges[1].Key},
			}))
			Expect(append(keysOf(all.Ranges), keysOf(rest.Ranges)...)).
				To(ConsistOf(byName))
		})

		DescribeTable("Should reject a request",
			func(ctx SpecContext, req apiranger.RetrieveRequest, message string) {
				Expect(retrieve(authed(ctx), req)).Error().To(SatisfyAll(
					BeAssignableToTypeOf(validate.PathError{}),
					MatchError(message),
				))
			},
			Entry(
				"that resumes an order without the order's value",
				apiranger.RetrieveRequest{
					Order: &apiranger.Order{Field: apiranger.OrderFieldName},
					After: &apiranger.Cursor{Key: uuid.New()},
				},
				"after.name: is required to resume an order by name: validation error",
			),
			Entry(
				"that resumes an order by end time without the end time",
				apiranger.RetrieveRequest{
					Order: &apiranger.Order{Field: apiranger.OrderFieldTimeRangeEnd},
					After: &apiranger.Cursor{Key: uuid.New()},
				},
				"after.time_range_end: is required to resume an order by "+
					"time_range_end: validation error",
			),
			Entry(
				"that sorts a search",
				apiranger.RetrieveRequest{
					SearchTerm: "fire",
					Order:      &apiranger.Order{Field: apiranger.OrderFieldName},
				},
				"order: a search sorts by relevance: validation error",
			),
			Entry(
				"that resumes a search",
				apiranger.RetrieveRequest{
					SearchTerm: "fire",
					After:      &apiranger.Cursor{Key: uuid.New()},
				},
				"after: a search cannot resume after a position: validation error",
			),
			Entry(
				"that sorts by an unknown field",
				apiranger.RetrieveRequest{Order: &apiranger.Order{Field: "color"}},
				`order.field: "color" is not a field to sort by: validation error`,
			),
		)
	})
})
