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
	"context"
	"encoding/json/v2"
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

var _ = Describe("Rename", func() {
	It("Should rename the range when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), r.ResourceID())
		rec := &Recorder[ranger.Key, ranger.Action]{}
		DeferCleanup(svc.Ranger.OnAction(rec.Record))
		MustSucceed(renameRange(
			AuthedCtx(ctx, u),
			apiranger.RenameRequest{Key: r.Key, Name: "Cold flow"},
		))
		Expect(retrieveRange(ctx, r.Key).Name).To(Equal("Cold flow"))
		Expect(rec.Snapshot()).To(HaveExactElements(HaveField("Key", r.Key)))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		Expect(renameRange(
			AuthedCtx(ctx, u),
			apiranger.RenameRequest{Key: r.Key, Name: "Cold flow"},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveRange(ctx, r.Key).Name).To(Equal("Hot fire"))
	})
})

var _ = Describe("SetEnd", func() {
	end := telem.TimeStamp(9 * telem.Second)
	It("Should set the end of the range when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), r.ResourceID())
		rec := &Recorder[ranger.Key, ranger.Action]{}
		DeferCleanup(svc.Ranger.OnAction(rec.Record))
		MustSucceed(setRangeEnd(
			AuthedCtx(ctx, u),
			apiranger.SetEndRequest{Key: r.Key, End: end},
		))
		Expect(retrieveRange(ctx, r.Key).TimeRange.End).To(Equal(end))
		Expect(rec.Snapshot()).To(HaveExactElements(HaveField("Key", r.Key)))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		Expect(setRangeEnd(
			AuthedCtx(ctx, u),
			apiranger.SetEndRequest{Key: r.Key, End: end},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveRange(ctx, r.Key).TimeRange).To(Equal(r.TimeRange))
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

	It("Should refuse to overwrite a range without update on it", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{
			Type: resource.TypeRange,
		})
		stored := createRange(ctx)
		overwrite := stored
		overwrite.Name = "Overwritten"
		Expect(create(AuthedCtx(ctx, u), apiranger.CreateRequest{
			Ranges: []ranger.Range{overwrite},
		})).Error().To(denied(u, access.ActionUpdate, stored.ResourceID()))
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
		"Should need update on the range to delete its key-value pairs with it",
		func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.KV.NewWriter(tx).SetMany(ctx, []kv.Pair{
					{Range: child.Key, Key: "k", Value: "v"},
				})
			})).To(Succeed())
			grant(ctx, access.ActionDelete, u.ResourceID(), childID())
			grant(ctx, access.ActionUpdate, u.ResourceID(), parentID())
			Expect(deleteChild(ctx)).Error().
				To(denied(u, access.ActionUpdate, childID()))
			Expect(rangeExists(ctx, child.Key)).To(BeTrue())
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
})
