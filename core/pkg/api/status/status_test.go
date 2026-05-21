// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// statusTypeOnly is the type-level ontology ID for granting access across all
// statuses. Row-level grants must match the input key exactly; type-level grants
// cover any input via the ontology.ID IsType matching rule.
var statusTypeOnly = ontology.ID{Type: ontology.ResourceTypeStatus}

var _ = Describe("api/status SetByKeyOrName", func() {
	Describe("authorized requests", func() {
		It("Should create a row with Key=Name=input when nothing matches", func(ctx SpecContext) {
			name := "api_set_fresh_" + uuid.New().String()
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionCreate},
				statusTypeOnly)

			res := MustSucceed(apiSvc.SetByKeyOrName(authedCtx(ctx, author), SetByKeyOrNameRequest{
				KeyOrName: name,
				Message:   "hello",
				Variant:   string(xstatus.VariantInfo),
			}))
			Expect(res.Key).To(Equal(name))
			Expect(res.MultipleMatches).To(BeFalse())

			var s status.Status[any]
			Expect(statusSvc.NewRetrieve().Where(status.MatchKeys[any](res.Key)).Entry(&s).Exec(ctx, nil)).To(Succeed())
			Expect(s.Key).To(Equal(name))
			Expect(s.Name).To(Equal(name))
			Expect(s.Message).To(Equal("hello"))
		})

		It("Should update an existing row when the input matches its Key", func(ctx SpecContext) {
			key := uuid.NewString()
			Expect(statusSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
				Key: key, Name: "api_uuid", Variant: xstatus.VariantInfo, Message: "orig", Time: telem.Now(),
			})).To(Succeed())
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionCreate},
				status.OntologyID(key))

			res := MustSucceed(apiSvc.SetByKeyOrName(authedCtx(ctx, author), SetByKeyOrNameRequest{
				KeyOrName: key,
				Message:   "updated",
				Variant:   string(xstatus.VariantWarning),
			}))
			Expect(res.Key).To(Equal(key))
			Expect(res.MultipleMatches).To(BeFalse())
		})

		It("Should report multipleMatches when the name resolves to multiple rows", func(ctx SpecContext) {
			name := "api_multi_" + uuid.New().String()
			Expect(statusSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
				Key: uuid.NewString(), Name: name, Variant: xstatus.VariantInfo, Message: "a", Time: telem.Now(),
			})).To(Succeed())
			Expect(statusSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
				Key: uuid.NewString(), Name: name, Variant: xstatus.VariantInfo, Message: "b", Time: telem.Now(),
			})).To(Succeed())
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionCreate},
				statusTypeOnly)

			res := MustSucceed(apiSvc.SetByKeyOrName(authedCtx(ctx, author), SetByKeyOrNameRequest{
				KeyOrName: name,
				Message:   "updated",
				Variant:   string(xstatus.VariantWarning),
			}))
			Expect(res.MultipleMatches).To(BeTrue())
		})
	})

	Describe("failure paths", func() {
		It("Should propagate ErrInvalidVariant", func(ctx SpecContext) {
			name := "api_iv_" + uuid.New().String()
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionCreate},
				statusTypeOnly)

			res, err := apiSvc.SetByKeyOrName(authedCtx(ctx, author), SetByKeyOrNameRequest{
				KeyOrName: name,
				Message:   "x",
				Variant:   "bogus",
			})
			Expect(err).To(MatchError(status.ErrInvalidVariant))
			Expect(res).To(Equal(SetByKeyOrNameResponse{}))
		})

		It("Should propagate ErrEmptyKeyOrName for empty input", func(ctx SpecContext) {
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionCreate},
				statusTypeOnly)

			res, err := apiSvc.SetByKeyOrName(authedCtx(ctx, author), SetByKeyOrNameRequest{
				KeyOrName: "",
				Message:   "x",
				Variant:   string(xstatus.VariantInfo),
			})
			Expect(err).To(MatchError(status.ErrEmptyKeyOrName))
			Expect(res).To(Equal(SetByKeyOrNameResponse{}))
		})

		It("Should refuse unauthorized requests without touching the store", func(ctx SpecContext) {
			name := "api_unauth_" + uuid.New().String()
			anon := freshUser(ctx)

			res, err := apiSvc.SetByKeyOrName(authedCtx(ctx, anon), SetByKeyOrNameRequest{
				KeyOrName: name,
				Message:   "noop",
				Variant:   string(xstatus.VariantInfo),
			})
			Expect(err).To(MatchError(access.ErrDenied))
			Expect(res).To(Equal(SetByKeyOrNameResponse{}))

			Expect(statusSvc.NewRetrieve().Where(status.MatchNames[any](name)).
				Entry(&status.Status[any]{}).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})

		It("Should deny when a row-level grant does not match the input key", func(ctx SpecContext) {
			name := "api_rowlevel_" + uuid.New().String()
			anon := freshUser(ctx)
			grantOn(ctx, user.OntologyID(anon.Key),
				[]access.Action{access.ActionCreate},
				status.OntologyID(uuid.NewString()))

			_, err := apiSvc.SetByKeyOrName(authedCtx(ctx, anon), SetByKeyOrNameRequest{
				KeyOrName: name,
				Message:   "x",
				Variant:   string(xstatus.VariantInfo),
			})
			Expect(err).To(MatchError(access.ErrDenied))
		})
	})
})

var _ = Describe("api/status DeleteByKeyOrName", func() {
	Describe("authorized requests", func() {
		It("Should delete a single by-name match", func(ctx SpecContext) {
			name := "api_del_single_" + uuid.New().String()
			Expect(statusSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
				Key: uuid.NewString(), Name: name, Variant: xstatus.VariantInfo, Message: "x", Time: telem.Now(),
			})).To(Succeed())
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionDelete},
				statusTypeOnly)

			res := MustSucceed(apiSvc.DeleteByKeyOrName(authedCtx(ctx, author), DeleteByKeyOrNameRequest{
				KeyOrName: name,
			}))
			Expect(res.Count).To(Equal(1))

			Expect(statusSvc.NewRetrieve().Where(status.MatchNames[any](name)).
				Entry(&status.Status[any]{}).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})

		It("Should report N when N rows share a name", func(ctx SpecContext) {
			name := "api_del_multi_" + uuid.New().String()
			Expect(statusSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
				Key: uuid.NewString(), Name: name, Variant: xstatus.VariantInfo, Message: "a", Time: telem.Now(),
			})).To(Succeed())
			Expect(statusSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
				Key: uuid.NewString(), Name: name, Variant: xstatus.VariantInfo, Message: "b", Time: telem.Now(),
			})).To(Succeed())
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionDelete},
				statusTypeOnly)

			res := MustSucceed(apiSvc.DeleteByKeyOrName(authedCtx(ctx, author), DeleteByKeyOrNameRequest{
				KeyOrName: name,
			}))
			Expect(res.Count).To(Equal(2))
		})

		It("Should return Count=0 when nothing matches", func(ctx SpecContext) {
			name := "api_del_missing_" + uuid.New().String()
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionDelete},
				statusTypeOnly)

			res := MustSucceed(apiSvc.DeleteByKeyOrName(authedCtx(ctx, author), DeleteByKeyOrNameRequest{
				KeyOrName: name,
			}))
			Expect(res.Count).To(Equal(0))
		})
	})

	Describe("failure paths", func() {
		It("Should refuse unauthorized requests without touching the store", func(ctx SpecContext) {
			name := "api_del_unauth_" + uuid.New().String()
			preKey := uuid.NewString()
			Expect(statusSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
				Key: preKey, Name: name, Variant: xstatus.VariantInfo, Message: "x", Time: telem.Now(),
			})).To(Succeed())
			anon := freshUser(ctx)

			res, err := apiSvc.DeleteByKeyOrName(authedCtx(ctx, anon), DeleteByKeyOrNameRequest{
				KeyOrName: name,
			})
			Expect(err).To(MatchError(access.ErrDenied))
			Expect(res).To(Equal(DeleteByKeyOrNameResponse{}))

			var s status.Status[any]
			Expect(statusSvc.NewRetrieve().Where(status.MatchKeys[any](preKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
		})

		It("Should propagate ErrEmptyKeyOrName for empty input", func(ctx SpecContext) {
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionDelete},
				statusTypeOnly)

			res, err := apiSvc.DeleteByKeyOrName(authedCtx(ctx, author), DeleteByKeyOrNameRequest{
				KeyOrName: "",
			})
			Expect(err).To(MatchError(status.ErrEmptyKeyOrName))
			Expect(res).To(Equal(DeleteByKeyOrNameResponse{}))
		})
	})
})
