// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/certificate"
	svccertificate "github.com/synnaxlabs/synnax/pkg/service/certificate"
	certificatepb "github.com/synnaxlabs/synnax/pkg/service/certificate/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	grpccertificate "github.com/synnaxlabs/synnax/pkg/transport/grpc/certificate"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var details = svccertificate.Details{
	Subject:   "CN=plant",
	Issuer:    "CN=plant",
	NotBefore: telem.TimeStamp(1),
	NotAfter:  telem.TimeStamp(2),
	Sha1:      "aa",
	Sha256:    "bb",
}

var _ = Describe("Set", func() {
	It("Should decode a file certificate input", func(ctx SpecContext) {
		var received certificate.SetRequest
		transport.CertificateSet.BindHandler(func(
			_ context.Context,
			req certificate.SetRequest,
		) (certificate.SetResponse, error) {
			received = req
			return certificate.SetResponse{}, nil
		})
		client := grpccertificate.NewCertificateSetServiceClient(conn)
		keyFile := "key.pem"
		MustSucceed(client.Exec(ctx, &grpccertificate.SetRequest{
			Certificates: []*grpccertificate.Input{{
				Name:            "Plant",
				Source:          string(secret.SourceFile),
				CertificatePath: "/etc/plant/cert.pem",
				PrivateKeyFile:  &keyFile,
				Details:         MustSucceed(certificatepb.DetailsToPB(details)),
				Rack:            3,
			}},
		}))
		Expect(received.Certificates).To(Equal([]certificate.Input{{
			Name:            "Plant",
			Source:          secret.SourceFile,
			CertificatePath: "/etc/plant/cert.pem",
			PrivateKeyFile:  &keyFile,
			Details:         &details,
			Rack:            rack.Key(3),
		}}))
	})
})

var _ = Describe("Refresh", func() {
	It("Should decode the key and the details", func(ctx SpecContext) {
		var received certificate.RefreshRequest
		transport.CertificateRefresh.BindHandler(func(
			_ context.Context,
			req certificate.RefreshRequest,
		) (struct{}, error) {
			received = req
			return struct{}{}, nil
		})
		client := grpccertificate.NewCertificateRefreshServiceClient(conn)
		key := uuid.New()
		MustSucceed(client.Exec(ctx, &grpccertificate.RefreshRequest{
			Key:     key.String(),
			Details: MustSucceed(certificatepb.DetailsToPB(details)),
		}))
		Expect(
			received,
		).To(Equal(certificate.RefreshRequest{Key: key, Details: details}))
	})
})
