// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification

import (
	"crypto/ed25519"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/synnaxlabs/x/encoding/base64"
	"github.com/synnaxlabs/x/errors"
)

// Anchors is the set of public keys a token may be signed under, keyed by the token
// header's key identifier.
type Anchors = map[string]ed25519.PublicKey

// anchors holds the production keys. Key 1 is the hub's signing key.
var anchors = Anchors{
	"1": ed25519.PublicKey(base64.MustDecode(
		"Sc6M2xG71KCrrL6XUjftfyLX8dLOSoBoN/LbXkT0lwE=",
	)),
}

// claimsVersion is the only claim set version this build understands.
const claimsVersion = 1

const headerKeyID = "kid"

// claims adapts Grant to the jwt.Claims interface. Validation of the times is the
// service's job, so the accessors only expose them.
type claims struct{ Grant }

var _ jwt.Claims = claims{}

func (c claims) GetExpirationTime() (*jwt.NumericDate, error) {
	if c.Exp == nil {
		return nil, nil
	}
	return jwt.NewNumericDate(time.Unix(int64(*c.Exp), 0)), nil
}

func (c claims) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(int64(c.Iat), 0)), nil
}

func (c claims) GetNotBefore() (*jwt.NumericDate, error) { return nil, nil }

func (c claims) GetIssuer() (string, error) { return "", nil }

func (c claims) GetSubject() (string, error) { return "", nil }

func (c claims) GetAudience() (jwt.ClaimStrings, error) { return nil, nil }

// Sign produces a token carrying g, signed with priv under the key identifier kid.
func Sign(priv ed25519.PrivateKey, kid string, g Grant) (string, error) {
	tk := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims{Grant: g})
	tk.Header[headerKeyID] = kid
	return tk.SignedString(priv)
}

// Verify checks token's signature against the anchor its header names and returns the
// grant it carries. It does not check the grant's term; the service does. Returns
// ErrInvalid on a bad signature, an unknown key, or an unsupported claim set version.
func Verify(anchors Anchors, token string) (Grant, error) {
	var c claims
	if _, err := jwt.ParseWithClaims(
		token,
		&c,
		func(t *jwt.Token) (any, error) {
			kid, _ := t.Header[headerKeyID].(string)
			key, ok := anchors[kid]
			if !ok {
				return nil, ErrInvalid
			}
			return key, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithoutClaimsValidation(),
	); err != nil {
		if errors.Is(err, ErrInvalid) {
			return Grant{}, err
		}
		return Grant{}, errors.Wrap(ErrInvalid, err.Error())
	}
	if c.V != claimsVersion {
		return Grant{}, errors.Wrapf(
			ErrInvalid,
			"unsupported claim set version %d",
			c.V,
		)
	}
	return c.Grant, nil
}
