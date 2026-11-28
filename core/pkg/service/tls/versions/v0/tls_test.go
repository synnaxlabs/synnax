// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/tls/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("TLS", func() {
	It("Should apply the defaults", func() {
		var t v0.TLS
		Expect(json.Unmarshal([]byte(`{}`), &t)).To(Succeed())
		Expect(t).To(Equal(v0.TLS{
			ServerTrust: v0.ServerTrust{Variant: v0.SystemRootsServerTrust{}},
		}))
	})

	DescribeTable("Should round trip a TLS configuration through JSON",
		func(t v0.TLS) {
			var fromJSON v0.TLS
			Expect(json.Unmarshal(MustSucceed(json.Marshal(t)), &fromJSON)).
				To(Succeed())
			Expect(fromJSON).To(Equal(t))
		},
		Entry("with the system roots", v0.TLS{
			ServerTrust: v0.ServerTrust{Variant: v0.SystemRootsServerTrust{}},
		}),
		Entry("with a certificate authority and a client certificate", v0.TLS{
			ServerTrust: v0.ServerTrust{Variant: v0.CertificateAuthorityServerTrust{
				Certificate: uuid.New(),
			}},
			ClientCertificate: new(uuid.New()),
		}),
		Entry("with validation disabled", v0.TLS{
			ServerTrust: v0.ServerTrust{Variant: v0.ValidationDisabledServerTrust{}},
		}),
	)
})
