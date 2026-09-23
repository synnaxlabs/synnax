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
	"crypto/ed25519"
	"crypto/rand"
	"time"
	"uuid"

	"github.com/samber/lo"
	distmock "github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/security"
	secmock "github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel/verification"
)

// KeyID is the anchor identifier Keys signs under.
const KeyID = "test"

// Keys is a throwaway signing key and the anchor set that verifies it.
type Keys struct {
	// Private signs grants.
	Private ed25519.PrivateKey
	// Anchors holds the matching public key under KeyID.
	Anchors verification.Anchors
}

// NewKeys generates a fresh signing key.
func NewKeys() Keys {
	pub, priv := lo.Must2(ed25519.GenerateKey(rand.Reader))
	return Keys{Private: priv, Anchors: verification.Anchors{KeyID: pub}}
}

// Sign signs g under the key.
func (k Keys) Sign(g verification.Grant) string {
	return lo.Must(verification.Sign(k.Private, KeyID, g))
}

// NewGrant returns a grant that floats between hosts and applies for fifty years.
func NewGrant() verification.Grant {
	now := time.Now()
	exp := uint32(now.Add(50 * 365 * 24 * time.Hour).Unix())
	return verification.Grant{
		Jti: uuid.New(),
		Iat: uint32(now.Unix()),
		Exp: &exp,
		V:   1,
		Org: uuid.New(),
		Ed:  "e",
		Fs:  1,
		N:   1,
	}
}

// OpenLayer opens a service layer on node with an insecure security provider and a
// grant that covers it. Fields set on cfgs take precedence.
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
	keys := NewKeys()
	base := service.LayerConfig{
		Distribution: node.Layer,
		Security:     sec,
		Storage:      node.Storage,
		Verifier:     keys.Sign(NewGrant()),
		Anchors:      keys.Anchors,
	}
	return service.OpenLayer(ctx, append([]service.LayerConfig{base}, cfgs...)...)
}
