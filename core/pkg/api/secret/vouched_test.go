// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apisecret "github.com/synnaxlabs/synnax/pkg/api/secret"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	certtestutil "github.com/synnaxlabs/synnax/pkg/service/certificate/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Vouched", func() {
	vouched := func(ctx SpecContext, target resource.ID) (bool, error) {
		GinkgoHelper()
		var (
			ok  bool
			err error
		)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			ok, err = apisecret.Vouched(svc.Certificate)(ctx, tx, target)
			return nil
		})).To(Succeed())
		return ok, err
	}
	It("Should vouch for a secret", func(ctx SpecContext) {
		s := createSecret(ctx, "Password", "hunter2")
		Expect(vouched(ctx, s.ResourceID())).To(BeTrue())
	})
	It("Should vouch for a certificate with a private key", func(ctx SpecContext) {
		Expect(vouched(ctx, createCertificate(ctx).ResourceID())).To(BeTrue())
	})
	It("Should not vouch for a certificate without a private key", func(
		ctx SpecContext,
	) {
		certPEM, _ := certtestutil.NewPair()
		c := certificate.Certificate{Variant: certificate.StoredCertificate{
			Base: certificate.Base{Name: "Server"},
			Pem:  certPEM,
		}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Certificate.NewWriter(tx).Set(ctx, &c, nil)
		})).To(Succeed())
		Expect(vouched(ctx, c.ResourceID())).To(BeFalse())
	})
	It("Should not vouch for a record of another type", func(ctx SpecContext) {
		Expect(vouched(ctx, resource.ID{Type: resource.TypeDevice, Key: "d1"})).
			To(BeFalse())
	})
	It("Should return query.ErrNotFound for a certificate that does not exist",
		func(ctx SpecContext) {
			Expect(vouched(ctx, certificate.ResourceID(uuid.New()))).
				Error().To(MatchError(query.ErrNotFound))
		},
	)
})
