// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tls_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	certtestutil "github.com/synnaxlabs/synnax/pkg/service/certificate/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/tls"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Certificates", func() {
	It("Should return no certificates for the system roots", func() {
		Expect(tls.Certificates(tls.TLS{})).To(BeEmpty())
	})

	It("Should return the authority and the client certificate", func() {
		ca, client := uuid.New(), uuid.New()
		t := tls.TLS{
			ServerTrust: tls.ServerTrust{Variant: tls.CertificateAuthorityServerTrust{
				Certificate: ca,
			}},
			ClientCertificate: &client,
		}
		Expect(tls.Certificates(t)).To(Equal([]certificate.Key{ca, client}))
	})
})

var _ = Describe("Validate", func() {
	var (
		tx    gorp.Tx
		certs *certificate.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roles := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		racks := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:      db,
			HostKey: 1,
			Status:  statuses,
			DriverRole: MustSucceed(
				racktestutil.CreateDriverRole(ctx, db, roles),
			),
			HealthCheckInterval: telem.Hour,
		}))
		secrets := secrettestutil.OpenService(
			ctx,
			db,
			secret.ServiceConfig{EncryptionKey: secret.NewEncryptionKey()},
		)
		certs = MustOpen(certificate.OpenService(ctx, certificate.ServiceConfig{
			DB:      db,
			Secret:  secrets,
			Status:  statuses,
			Rack:    racks.EmbeddedKey,
			HostKey: 1,
		}))
		tx = DeferClose(db.OpenTx())
	})

	createCertificate := func(ctx context.Context, privateKey bool) certificate.Key {
		GinkgoHelper()
		certPEM, keyPEM := certtestutil.NewPair()
		var key []byte
		if privateKey {
			key = []byte(keyPEM)
		}
		c := certificate.Certificate{Variant: certificate.StoredCertificate{
			Base: certificate.Base{Name: "Broker"},
			Pem:  certPEM,
		}}
		Expect(certs.NewWriter(tx).Set(ctx, &c, key)).To(Succeed())
		return c.Base().Key
	}

	It("Should accept no client certificate", func(ctx SpecContext) {
		Expect(tls.Validate(ctx, tx, certs, tls.TLS{})).To(Succeed())
	})

	It("Should accept a client certificate with a private key", func(ctx SpecContext) {
		key := createCertificate(ctx, true)
		Expect(tls.Validate(ctx, tx, certs, tls.TLS{ClientCertificate: &key})).
			To(Succeed())
	})

	It("Should refuse a client certificate with no private key", func(ctx SpecContext) {
		key := createCertificate(ctx, false)
		Expect(tls.Validate(ctx, tx, certs, tls.TLS{ClientCertificate: &key})).
			To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("client_certificate")),
				MatchError(ContainSubstring("certificate has no private key")),
			))
	})
})
