// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package license

import (
	"crypto/mldsa"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Anchors is the set of public keys a token may be signed under, keyed by the token
// header's key identifier.
type Anchors = map[string]*mldsa.PublicKey

// anchors holds the production keys. Key 2 is the portal's ML-DSA-44 signing key; its
// private half lives only in KMS.
var anchors = Anchors{"2": mustPublicKey(
	"FAWBSKFJXIX79I7MUoSMyD+FbKhEuDFhX0cem9a4agSss+xl0DabFEkuKi7HiEzyYM/GxaG+" +
		"Aph+dNgyjlejcDNWZCGkh+byD93ZCPYl1EbV26JyCH0ufVEEVPLoENmrgILHnLVn4yyrRmt1" +
		"dSOjKqfGSqI1uCFCI8rbQt+T/wRkwA4D0CTxDzuI/dWYyUIZEKyTE118mPrWi6W8A9A4yR6N" +
		"1/cVY3HzWckHQNfbJHfelNcc5WuVZtLKiAZ8boKn/PJ1MuTps2QCzUnM5HVkVcQDjaClMX47" +
		"jyydj7jczT0ks4XR9TYnmr7tGl/ww53QW9o+tONc78R2qu/LOkDd0uVJcWsAOPIxLERmxSyW" +
		"pgr3VGamCPKSGAMgaXfWtQsWCedlagly1kwfRlFuUa4ZJE1f4IAH8UB84q7zina8o2kq+6ur" +
		"i7lQfd4hs1Yn1RKY8zc1Y17tUNQG7MeULrQe2lL6/9hLx0Gwkh43C9rizYKjLhlzDxM+bBaL" +
		"BPkNNoa34Mb9300e8BBrSHgahNxYIxIFidZdch7zaAx5AvyusHsLrR/bqyWoKDQx+GbodgQM" +
		"fq4DA7DBtkKpN6UtipfQanUxtpchgJytjd2e9xx2654+ABb1gzEMAzpNT0jeuyYEqTJHxnRq" +
		"gMtY2xcLX4/LdBRA+FucPUzNW3ufxvvq7ds5RjilfGnegrd6pC5OVkbRk0/6ZE0SBVRPB6lx" +
		"n0CTVb19b7hyKzJ6OxiwLCYM+N1ffyJ5LmfT5JoB84cQbFwNFsCT74bge/fXQv7ignTP1JgU" +
		"t1bziLMMmxwWJT4LWLdfTSFRtcKB8sedbEe0X1N0WU/Egg8QYl0z3V5tTdFsi+o6ekAougoY" +
		"Okh30QL6E8Xxx40xr0abf6lpB6GOffr8vBlB/xiWZxjx1XbwyXscY9Dt6Nmbr5kqWFypo0Ox" +
		"QS2bp0+0WzL0hQy8Y+BvQcyJYoxwC8/UEJMIIL1GtphaabpuZoXgwzda5aK4puXdVSfyF7+9" +
		"CCL8gtRXPNBbp6POzkJIY+vMz1EARt7fgYcY0kpShW1e8WkEd0sUS268Aa/2vN5MrjgqR6RY" +
		"+4AvU02cXQ83ClJP+0uBO1dPLtnStIoR2lli1FAxBLvOXe16MeniSIPzRPIglITRsz6GTOBI" +
		"WVKIQynKFAg163HI2sgbDOa0UYNyy1jBh+6k+XXz+yuyWXMKdpxoHxKMYmsvGAsaDr4tP7tm" +
		"QtSs/ezxocazvL2TnPuEZOwv+LPJotwZ+iGrTZ/ngwx/gh+yAiVubOsp3imRjSqTFIt9EhR1" +
		"K7bdVxPBsbfbJO1ARlIu52UZW/lrXg9dqUZSKVeUF440mGujdIF0uWG6mXBHfnHQ9gMXlDuF" +
		"Isam2OIZbRJw9uAP0q3RbZDRBwIp6WaVeci87QbCtRKRVrHVHxsTnaTSu73qxtXUfEvwufDp" +
		"kmqF7quU/W2X4cEo1mT3bqaS5ljB6k/fmJxvTELjxwVt+Est+Gnfvc+LGk0qgvB82Gq8U9fF" +
		"c017jz5TNJH4v46gzje+bM5Q9fkK58S0Chwclb8B/1N08DQBXTiuiGWTJVVEYX/sB8p+NNXq" +
		"AY18QIRrIC1uBqMWSf6+vdDMXbZR0pAmYqfU1zDWxFY45QBOCKyikIchd/KJn4r48qln/twt" +
		"8BL4pYMYyFbJw5pFLChRYa/g7t+eWokk+jq6OGNTOoZJhOqywaV4DNX0qP3LNamAAFBRHqh2" +
		"yfCDm9zfYWvq1c+YdJYLYQ==",
)}

func mustPublicKey(encoded string) *mldsa.PublicKey {
	return lo.Must(mldsa.NewPublicKey(
		mldsa.MLDSA44(),
		lo.Must(base64.StdEncoding.DecodeString(encoded)),
	))
}

// algorithm is the JWS algorithm every token is signed with.
const algorithm = "ML-DSA-44"

// claimsVersion is the only claim set version this build understands.
const claimsVersion = 1

// understood is every claim this build reads. A token whose required list names a claim
// outside it is refused, so a Core never ignores a rule it cannot enforce.
var understood = set.New(
	"jti",
	"iat",
	"exp",
	"claims_version",
	"organization",
	"edition",
	"fingerprints",
	"fingerprint_scheme",
	"machines",
	"channels",
	"max_version",
	"required",
)

// header is the JWS protected header of a token.
type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

var rawURL = base64.RawURLEncoding

// Sign produces a token carrying lic, signed with priv under the key identifier kid.
func Sign(priv *mldsa.PrivateKey, kid string, lic License) (string, error) {
	h, err := json.Marshal(header{Alg: algorithm, Typ: "JWT", Kid: kid})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(lic)
	if err != nil {
		return "", err
	}
	input := rawURL.EncodeToString(h) + "." + rawURL.EncodeToString(payload)
	sig, err := priv.Sign(nil, []byte(input), nil)
	if err != nil {
		return "", err
	}
	return input + "." + rawURL.EncodeToString(sig), nil
}

// Verify checks token's signature against the anchor its header names and returns the
// license it carries. It does not check the license's term; the service does. Returns
// ErrInvalid on a malformed token, a bad signature, an unknown key, an unsupported
// claim set version, a required claim this Core does not understand, a missing or
// unparseable version ceiling on a license without an expiry, or an unparseable
// ceiling on any license.
func Verify(anchors Anchors, token string) (License, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return License{}, errors.Wrap(ErrInvalid, "a token has three parts")
	}
	var h header
	if err := decodePart(parts[0], &h); err != nil {
		return License{}, err
	}
	if h.Alg != algorithm {
		return License{}, errors.Wrapf(ErrInvalid, "unsupported algorithm %q", h.Alg)
	}
	key, ok := anchors[h.Kid]
	if !ok {
		return License{}, errors.Wrapf(ErrInvalid, "unknown key %q", h.Kid)
	}
	sig, err := rawURL.DecodeString(parts[2])
	if err != nil {
		return License{}, errors.Wrap(ErrInvalid, err.Error())
	}
	input := parts[0] + "." + parts[1]
	if err = mldsa.Verify(key, []byte(input), sig, nil); err != nil {
		return License{}, errors.Wrap(ErrInvalid, err.Error())
	}
	var c License
	if err = decodePart(parts[1], &c); err != nil {
		return License{}, err
	}
	if c.ClaimsVersion != claimsVersion {
		return License{}, errors.Wrapf(
			ErrInvalid,
			"unsupported claim set version %d",
			c.ClaimsVersion,
		)
	}
	if err := c.Validate(); err != nil {
		return License{}, errors.Wrap(ErrInvalid, err.Error())
	}
	unknown := lo.Filter(c.Required, func(claim string, _ int) bool {
		return !understood.Contains(claim)
	})
	if len(unknown) > 0 {
		return License{}, errors.Wrapf(
			ErrInvalid,
			"the license needs a newer Core that understands %s",
			strings.Join(unknown, ", "),
		)
	}
	if c.Exp == nil && c.MaxVersion == nil {
		return License{}, errors.Wrap(
			ErrInvalid,
			"a license without an expiry must carry a maximum version",
		)
	}
	if c.MaxVersion != nil {
		if _, _, ok := parseMinor(*c.MaxVersion); !ok {
			return License{}, errors.Wrapf(
				ErrInvalid,
				"bad version ceiling %q",
				*c.MaxVersion,
			)
		}
	}
	return c, nil
}

func decodePart(part string, v any) error {
	b, err := rawURL.DecodeString(part)
	if err != nil {
		return errors.Wrap(ErrInvalid, err.Error())
	}
	if err = json.Unmarshal(b, v); err != nil {
		return errors.Wrap(ErrInvalid, err.Error())
	}
	return nil
}
