// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package license_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apilicense "github.com/synnaxlabs/synnax/pkg/api/license"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	license "github.com/synnaxlabs/synnax/pkg/service/channel/license"
	svcmock "github.com/synnaxlabs/synnax/pkg/service/mock"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
)

var object = ontology.ID{Type: ontology.ResourceTypeLicense}

var _ = Describe("Service", Ordered, func() {
	It("Should refuse retrieval without a retrieve license", func(ctx SpecContext) {
		Expect(apiSvc.Retrieve(
			AuthedCtx(ctx, freshUser(ctx)),
			apilicense.RetrieveRequest{},
		)).Error().To(MatchError(access.ErrDenied))
	})

	It("Should report a missing license to a reader", func(ctx SpecContext) {
		reader := freshUser(ctx)
		grantOn(
			ctx,
			reader.OntologyID(),
			[]access.Action{access.ActionRetrieve},
			object,
		)
		info := MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, reader),
			apilicense.RetrieveRequest{},
		))
		Expect(info.State).To(Equal(license.StateMissing))
		Expect(info.License).To(BeNil())
	})

	It("Should gate requests while the license is missing", func(ctx SpecContext) {
		called := false
		// Exec hands the context back beside the error, so .Error() cannot apply.
		_, err := apilicense.Middleware(licenseSvc).Exec(
			freighter.Context{Context: ctx},
			func(fctx freighter.Context) (freighter.Context, error) {
				called = true
				return fctx, nil
			},
		)
		Expect(err).To(MatchError(license.ErrMissing))
		Expect(called).To(BeFalse())
	})

	It("Should refuse a token without an update license", func(ctx SpecContext) {
		reader := freshUser(ctx)
		grantOn(
			ctx,
			reader.OntologyID(),
			[]access.Action{access.ActionRetrieve},
			object,
		)
		Expect(apiSvc.Apply(
			AuthedCtx(ctx, reader),
			apilicense.ApplyRequest{Token: keys.Sign(svcmock.NewLicense())},
		)).Error().To(MatchError(access.ErrDenied))
	})

	It("Should apply a token for an owner", func(ctx SpecContext) {
		owner := freshUser(ctx)
		grantOn(ctx, owner.OntologyID(), []access.Action{access.ActionUpdate}, object)
		lic := svcmock.NewLicense()
		info := MustSucceed(apiSvc.Apply(
			AuthedCtx(ctx, owner),
			apilicense.ApplyRequest{Token: keys.Sign(lic)},
		))
		Expect(info.State).To(Equal(license.StateOK))
		Expect(info.License).ToNot(BeNil())
		Expect(info.License.Jti).To(Equal(lic.Jti))
	})

	It("Should pass requests once a license applies", func(ctx SpecContext) {
		called := false
		MustSucceed(apilicense.Middleware(licenseSvc).Exec(
			freighter.Context{Context: ctx},
			func(fctx freighter.Context) (freighter.Context, error) {
				called = true
				return fctx, nil
			},
		))
		Expect(called).To(BeTrue())
	})
})
