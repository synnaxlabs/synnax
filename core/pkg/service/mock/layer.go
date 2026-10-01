// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package mock opens service layers for tests.
package mock

import (
	"context"
	"crypto/mldsa"
	"time"
	"uuid"

	"github.com/samber/lo"
	distmock "github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/security"
	secmock "github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/license"
)

const keyID = "test"

// Signer is a throwaway signing key and the anchor set that verifies it.
type Signer struct {
	private *mldsa.PrivateKey
	// Anchors holds the matching public key.
	Anchors license.Anchors
}

// NewSigner generates a fresh signing key.
func NewSigner() Signer {
	priv := lo.Must(mldsa.GenerateKey(mldsa.MLDSA44()))
	return Signer{private: priv, Anchors: license.Anchors{keyID: priv.PublicKey()}}
}

// Sign returns a license key carrying lic.
func (s Signer) Sign(lic license.License) string {
	return lo.Must(license.Sign(s.private, keyID, lic))
}

// NewLicense returns a license that floats between machines and lasts fifty years.
func NewLicense() license.License {
	now := time.Now()
	exp := uint32(now.Add(50 * 365 * 24 * time.Hour).Unix())
	return license.License{
		Jti:               uuid.New(),
		Iat:               uint32(now.Unix()),
		Exp:               &exp,
		ClaimsVersion:     1,
		Organization:      uuid.New(),
		Edition:           license.EditionEnterprise,
		FingerprintScheme: 1,
		Machines:          1,
	}
}

// OpenLayer opens a service layer on node with an insecure security provider and a
// license that covers it. Fields set on cfgs take precedence.
func OpenLayer(
	ctx context.Context,
	node distmock.Node,
	cfgs ...service.LayerConfig,
) (*service.Layer, error) {
	sec, err := security.NewProvider(security.ProviderConfig{
		Insecure: new(true),
		KeySize:  secmock.SmallKeySize,
	})
	if err != nil {
		return nil, err
	}
	signer := NewSigner()
	base := service.LayerConfig{
		Distribution: node.Layer,
		Security:     sec,
		Storage:      node.Storage,
		License: license.ServiceConfig{
			Key:     signer.Sign(NewLicense()),
			Anchors: signer.Anchors,
		},
	}
	return service.OpenLayer(ctx, append([]service.LayerConfig{base}, cfgs...)...)
}
