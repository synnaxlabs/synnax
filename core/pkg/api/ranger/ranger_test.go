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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	apiranger "github.com/synnaxlabs/synnax/pkg/api/ranger"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
		Expect(svc.Ranger.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), r.ResourceID())
		res := MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, u),
			apiranger.RetrieveRequest{Where: &ranger.Where{
				Key: &filter.Set[apiranger.Key]{In: []apiranger.Key{r.Key, uuid.New()}},
			}},
		))
		Expect(res.Ranges).To(HaveExactElements(HaveField("Key", r.Key)))
	})

	It("Should return no range for a zero limit", func(ctx SpecContext) {
		u := createUser(ctx)
		r := apiranger.Range{
			Key:       uuid.New(),
			Name:      "limited",
			TimeRange: telem.TimeRange{Start: 1, End: 2},
		}
		Expect(svc.Ranger.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), r.ResourceID())
		res := MustSucceed(apiSvc.Retrieve(
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
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apiranger.RetrieveRequest{
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
	Expect(svc.Ranger.NewWriter(nil).Create(ctx, &r)).To(Succeed())
	return r
}

func retrieveRange(ctx SpecContext, key ranger.Key) ranger.Range {
	GinkgoHelper()
	var r ranger.Range
	Expect(svc.Ranger.NewRetrieve().
		Where(ranger.MatchKeys(key)).
		Entry(&r).
		Exec(ctx, nil)).To(Succeed())
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
		MustSucceed(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			apiranger.DispatchRequest{Key: r.Key, DispatchKey: "dk", Actions: rename},
		))
		Expect(retrieveRange(ctx, r.Key).Name).To(Equal("Cold flow"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		Expect(apiSvc.Dispatch(
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
			Expect(apiSvc.Dispatch(
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
			Expect(svc.Ranger.NewWriter(nil).Create(ctx, &child)).To(Succeed())
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				child.ResourceID(),
				newParent.ResourceID(),
			)
			Expect(apiSvc.Dispatch(
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
		Expect(svc.Ranger.NewWriter(nil).Create(ctx, &child)).To(Succeed())
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			child.ResourceID(),
			oldParent.ResourceID(),
			newParent.ResourceID(),
		)
		MustSucceed(apiSvc.Dispatch(
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
		Expect(apiSvc.Dispatch(
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
				apiSvc.Create(AuthedCtx(ctx, u), nil, apiranger.CreateRequest{
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
		Expect(svc.Ranger.NewWriter(nil).Create(ctx, &child)).To(Succeed())
		createRange(ctx)
		res := MustSucceed(apiSvc.Retrieve(authed(ctx), apiranger.RetrieveRequest{
			Where: &ranger.Where{Parents: []ranger.Key{parent.Key}},
		}))
		Expect(keysOf(res.Ranges)).To(Equal([]ranger.Key{child.Key}))
		Expect(res.Ranges[0].Parent).To(HaveValue(Equal(parent.Key)))
	})

	It("Should retrieve the ranges that hold any of the labels", func(ctx SpecContext) {
		a, b := label.Label{Name: "A"}, label.Label{Name: "B"}
		Expect(svc.Label.NewWriter(nil).Create(ctx, &a)).To(Succeed())
		Expect(svc.Label.NewWriter(nil).Create(ctx, &b)).To(Succeed())
		withA, withB := createRange(ctx), createRange(ctx)
		createRange(ctx)
		Expect(
			svc.Ranger.NewWriter(nil).
				AddLabels(ctx, withA.Key, []label.Key{a.Key}),
		).
			To(Succeed())
		Expect(
			svc.Ranger.NewWriter(nil).
				AddLabels(ctx, withB.Key, []label.Key{b.Key}),
		).
			To(Succeed())
		res := MustSucceed(apiSvc.Retrieve(authed(ctx), apiranger.RetrieveRequest{
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
		)
		BeforeEach(func(ctx SpecContext) {
			parent = createRange(ctx)
			child := func(name string, start telem.TimeStamp) ranger.Key {
				GinkgoHelper()
				r := ranger.Range{
					Name:      name,
					TimeRange: telem.TimeRange{Start: start, End: start + 1},
					Parent:    &parent.Key,
				}
				Expect(svc.Ranger.NewWriter(nil).Create(ctx, &r)).To(Succeed())
				return r.Key
			}
			beta := child("beta", 30)
			upper := child("Alpha", 10)
			gamma := child("Gamma", 20)
			lower := child("alpha", 40)
			tie := []ranger.Key{upper, lower}
			// Gorp orders equal names by the trailing bytes of the key.
			slices.SortFunc(tie, func(a, b ranger.Key) int {
				slices.Reverse(a[:])
				slices.Reverse(b[:])
				return bytes.Compare(a[:], b[:])
			})
			byName = append(tie, beta, gamma)
			byStart = []ranger.Key{lower, beta, gamma, upper}
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
				res := MustSucceed(apiSvc.Retrieve(ctx, apiranger.RetrieveRequest{
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

		It("Should page ranges by key with no order", func(ctx SpecContext) {
			authedCtx := authed(ctx)
			all := MustSucceed(apiSvc.Retrieve(authedCtx, apiranger.RetrieveRequest{
				Where: &ranger.Where{Parents: []ranger.Key{parent.Key}},
				Limit: new(uint32(2)),
			}))
			Expect(all.Ranges).To(HaveLen(2))
			rest := MustSucceed(apiSvc.Retrieve(authedCtx, apiranger.RetrieveRequest{
				Where: &ranger.Where{Parents: []ranger.Key{parent.Key}},
				After: &apiranger.Cursor{Key: all.Ranges[1].Key},
			}))
			Expect(append(keysOf(all.Ranges), keysOf(rest.Ranges)...)).
				To(ConsistOf(byName))
		})

		DescribeTable("Should reject a request",
			func(ctx SpecContext, req apiranger.RetrieveRequest, message string) {
				Expect(apiSvc.Retrieve(authed(ctx), req)).Error().To(SatisfyAll(
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
