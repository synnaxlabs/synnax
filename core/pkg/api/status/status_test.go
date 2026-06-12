// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// statusTypeOnly is the type-level ontology ID for granting access across all
// statuses. Row-level grants must match the input key exactly; type-level grants
// cover any input via the ontology.ID IsType matching rule.
var statusTypeOnly = ontology.ID{Type: ontology.ResourceTypeStatus}

var _ = Describe("api/status SetByKeyOrName", func() {
	Describe("authorized requests", func() {
		It("Should create a UUID-keyed row named after the input when nothing matches", func(ctx SpecContext) {
			name := "api_set_fresh_" + uuid.New().String()
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionCreate},
				statusTypeOnly)

			res := MustSucceed(apiSvc.SetByKeyOrName(authedCtx(ctx, author), db, apistatus.SetByKeyOrNameRequest{
				KeyOrName: name,
				Message:   "hello",
				Variant:   xstatus.VariantInfo,
			}))
			MustSucceed(uuid.Parse(res.Key))
			Expect(res.Key).ToNot(Equal(name))
			Expect(res.MultipleMatches).To(BeFalse())

			var s status.Status[any]
			Expect(statusSvc.NewRetrieve().Where(status.MatchKeys[any](res.Key)).Entry(&s).Exec(ctx, nil)).To(Succeed())
			Expect(s.Key).To(Equal(res.Key))
			Expect(s.Name).To(Equal(name))
			Expect(s.Message).To(Equal("hello"))
		})

		It("Should update an existing row when the input matches its Key", func(ctx SpecContext) {
			key := uuid.NewString()
			Expect(statusSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
				Status: xstatus.Status[any]{Variant: xstatus.VariantInfo, Message: "orig", Time: telem.Now()},
				Key:    key, Name: "api_uuid",
			})).To(Succeed())
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionUpdate},
				status.OntologyID(key))

			res := MustSucceed(apiSvc.SetByKeyOrName(authedCtx(ctx, author), db, apistatus.SetByKeyOrNameRequest{
				KeyOrName: key,
				Message:   "updated",
				Variant:   xstatus.VariantWarning,
			}))
			Expect(res.Key).To(Equal(key))
			Expect(res.MultipleMatches).To(BeFalse())
		})

		It("Should report multipleMatches when the name resolves to multiple rows", func(ctx SpecContext) {
			name := "api_multi_" + uuid.New().String()
			Expect(statusSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
				Status: xstatus.Status[any]{Variant: xstatus.VariantInfo, Message: "a", Time: telem.Now()},
				Key:    uuid.NewString(), Name: name,
			})).To(Succeed())
			Expect(statusSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
				Status: xstatus.Status[any]{Variant: xstatus.VariantInfo, Message: "b", Time: telem.Now()},
				Key:    uuid.NewString(), Name: name,
			})).To(Succeed())
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionUpdate},
				statusTypeOnly)

			res := MustSucceed(apiSvc.SetByKeyOrName(authedCtx(ctx, author), db, apistatus.SetByKeyOrNameRequest{
				KeyOrName: name,
				Message:   "updated",
				Variant:   xstatus.VariantWarning,
			}))
			Expect(res.MultipleMatches).To(BeTrue())
		})
	})

	Describe("failure paths", func() {
		It("Should propagate a validation error for an unknown variant", func(ctx SpecContext) {
			name := "api_iv_" + uuid.New().String()
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionCreate},
				statusTypeOnly)

			Expect(apiSvc.SetByKeyOrName(authedCtx(ctx, author), db, apistatus.SetByKeyOrNameRequest{
				KeyOrName: name,
				Message:   "x",
				Variant:   "bogus",
			})).Error().To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("invalid status variant"))))
		})

		It("Should propagate a validation error for empty input", func(ctx SpecContext) {
			grantOn(ctx, user.OntologyID(author.Key),
				[]access.Action{access.ActionCreate},
				statusTypeOnly)

			Expect(apiSvc.SetByKeyOrName(authedCtx(ctx, author), db, apistatus.SetByKeyOrNameRequest{
				KeyOrName: "",
				Message:   "x",
				Variant:   xstatus.VariantInfo,
			})).Error().To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("key_or_name is required"))))
		})

		It("Should refuse unauthorized requests without touching the store", func(ctx SpecContext) {
			name := "api_unauth_" + uuid.New().String()
			anon := freshUser(ctx)

			Expect(apiSvc.SetByKeyOrName(authedCtx(ctx, anon), db, apistatus.SetByKeyOrNameRequest{
				KeyOrName: name,
				Message:   "noop",
				Variant:   xstatus.VariantInfo,
			})).Error().To(MatchError(access.ErrDenied))

			Expect(statusSvc.NewRetrieve().Where(status.MatchKeys[any](name)).
				Entry(&status.Status[any]{}).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})

		It("Should deny when a row-level grant does not match the input key", func(ctx SpecContext) {
			name := "api_rowlevel_" + uuid.New().String()
			anon := freshUser(ctx)
			grantOn(ctx, user.OntologyID(anon.Key),
				[]access.Action{access.ActionCreate},
				status.OntologyID(uuid.NewString()))

			Expect(apiSvc.SetByKeyOrName(authedCtx(ctx, anon), db, apistatus.SetByKeyOrNameRequest{
				KeyOrName: name,
				Message:   "x",
				Variant:   xstatus.VariantInfo,
			})).Error().To(MatchError(access.ErrDenied))
		})
	})
})
