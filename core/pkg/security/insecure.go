// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package security

import (
	"crypto/tls"

	"github.com/synnaxlabs/synnax/pkg/security/cert"
)

type insecureProvider struct{}

func newInsecureProvider(ProviderConfig) (Provider, error) {
	return &insecureProvider{}, nil
}

func (*insecureProvider) TLSConfigFor(cert.Source) *tls.Config { return nil }

func (*insecureProvider) NodeClientConfig() *tls.Config { return nil }

func (*insecureProvider) VerifyCertHost(cert.Source, string) error { return nil }

func (*insecureProvider) VerifyCertCoreCA(cert.Source) error { return nil }

func (*insecureProvider) VerifyCertTrustAnchors(cert.Source) error { return nil }
