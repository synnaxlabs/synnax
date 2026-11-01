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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiranger "github.com/synnaxlabs/synnax/pkg/api/ranger"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
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
		grant(ctx, access.ActionRetrieve, u.OntologyID(), r.OntologyID())
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
		grant(ctx, access.ActionRetrieve, u.OntologyID(), r.OntologyID())
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
		grant(ctx, access.ActionUpdate, u.OntologyID(), r.OntologyID())
		MustSucceed(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apiranger.DispatchRequest{Key: r.Key, DispatchKey: "dk", Actions: rename},
		))
		Expect(retrieveRange(ctx, r.Key).Name).To(Equal("Cold flow"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apiranger.DispatchRequest{Key: r.Key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveRange(ctx, r.Key).Name).To(Equal("Hot fire"))
	})
	It("Should return the dispatch error", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.OntologyID(), ranger.OntologyID(key))
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apiranger.DispatchRequest{Key: key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Create", func() {
	It(
		"Should stamp the dispatch key onto the create broadcast",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, access.ActionCreate, u.OntologyID(), ontology.ID{
				Type: ontology.ResourceTypeRange,
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
