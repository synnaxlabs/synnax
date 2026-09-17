// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	apiverification "github.com/synnaxlabs/synnax/pkg/api/verification"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	svcverification "github.com/synnaxlabs/synnax/pkg/service/channel/verification"
	svcmock "github.com/synnaxlabs/synnax/pkg/service/mock"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
)

var object = ontology.ID{Type: ontology.ResourceTypeVerification}

var _ = Describe("Service", Ordered, func() {
	It("Should refuse retrieval without a retrieve grant", func(ctx SpecContext) {
		_, err := apiSvc.Retrieve(
			AuthedCtx(ctx, freshUser(ctx)),
			apiverification.RetrieveRequest{},
		)
		Expect(err).To(MatchError(access.ErrDenied))
	})

	It("Should report a missing grant to a reader", func(ctx SpecContext) {
		reader := freshUser(ctx)
		grantOn(ctx, reader.OntologyID(), []access.Action{access.ActionRetrieve}, object)
		info := MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, reader),
			apiverification.RetrieveRequest{},
		))
		Expect(info.State).To(Equal(svcverification.StateMissing))
		Expect(info.Grant).To(BeNil())
	})

	It("Should gate requests while the grant is missing", func(ctx SpecContext) {
		called := false
		_, err := apiverification.Middleware(verSvc).Exec(
			freighter.Context{Context: ctx},
			func(fctx freighter.Context) (freighter.Context, error) {
				called = true
				return fctx, nil
			},
		)
		Expect(err).To(MatchError(svcverification.ErrMissing))
		Expect(called).To(BeFalse())
	})

	It("Should refuse activation without an update grant", func(ctx SpecContext) {
		reader := freshUser(ctx)
		grantOn(ctx, reader.OntologyID(), []access.Action{access.ActionRetrieve}, object)
		_, err := apiSvc.Activate(
			AuthedCtx(ctx, reader),
			apiverification.ActivateRequest{Token: keys.Sign(svcmock.NewGrant())},
		)
		Expect(err).To(MatchError(access.ErrDenied))
	})

	It("Should activate a token for an owner", func(ctx SpecContext) {
		owner := freshUser(ctx)
		grantOn(ctx, owner.OntologyID(), []access.Action{access.ActionUpdate}, object)
		grant := svcmock.NewGrant()
		info := MustSucceed(apiSvc.Activate(
			AuthedCtx(ctx, owner),
			apiverification.ActivateRequest{Token: keys.Sign(grant)},
		))
		Expect(info.State).To(Equal(svcverification.StateOK))
		Expect(info.Grant).ToNot(BeNil())
		Expect(info.Grant.Jti).To(Equal(grant.Jti))
	})

	It("Should pass requests once a grant applies", func(ctx SpecContext) {
		called := false
		MustSucceed(apiverification.Middleware(verSvc).Exec(
			freighter.Context{Context: ctx},
			func(fctx freighter.Context) (freighter.Context, error) {
				called = true
				return fctx, nil
			},
		))
		Expect(called).To(BeTrue())
	})
})
