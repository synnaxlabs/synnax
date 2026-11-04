// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"slices"
	"strings"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

type grantClaims struct {
	Rack    string `json:"rack"`
	Secrets []Key  `json:"secrets"`
	Expires int64  `json:"expires"`
}

// Grant returns a grant that lets rack read the secrets with the given keys until
// GrantTTL passes, whether or not anything on the rack uses them. It returns
// query.ErrNotFound when a secret does not exist.
func (s *Service) Grant(
	ctx context.Context,
	tx gorp.Tx,
	rack ontology.ID,
	keys ...Key,
) (string, error) {
	v := validate.New("secret.grant")
	v.Ternaryf("rack", rack.Type != ontology.ResourceTypeRack, "%s is not a rack", rack)
	v.NotEmptySlice("secrets", keys)
	if err := v.Error(); err != nil {
		return "", err
	}
	if err := s.NewRetrieve().Where(MatchKeys(keys...)).Exec(ctx, tx); err != nil {
		return "", err
	}
	claims, err := json.Marshal(grantClaims{
		Rack:    rack.String(),
		Secrets: keys,
		Expires: int64(s.cfg.Now().Add(s.cfg.GrantTTL)),
	})
	if err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding
	return enc.EncodeToString(claims) + "." + enc.EncodeToString(s.sign(claims)), nil
}

func (s *Service) sign(claims []byte) []byte {
	mac := hmac.New(sha256.New, s.grantKey)
	mac.Write(claims)
	return mac.Sum(nil)
}

var errInvalidGrant = errors.Wrap(access.ErrDenied, "grant is not valid")

// granted returns the secrets that grant lets rack read. An empty grant lets rack read
// nothing.
func (s *Service) granted(grant string, rack ontology.ID) (set.Set[Key], error) {
	if grant == "" {
		return nil, nil
	}
	enc := base64.RawURLEncoding
	encodedClaims, encodedSig, found := strings.Cut(grant, ".")
	if !found {
		return nil, errInvalidGrant
	}
	claims, err := enc.DecodeString(encodedClaims)
	if err != nil {
		return nil, errInvalidGrant
	}
	sig, err := enc.DecodeString(encodedSig)
	if err != nil || !hmac.Equal(sig, s.sign(claims)) {
		return nil, errInvalidGrant
	}
	var c grantClaims
	if err = json.Unmarshal(claims, &c); err != nil {
		return nil, errInvalidGrant
	}
	if c.Rack != rack.String() {
		return nil, errors.Wrapf(access.ErrDenied, "grant is not for %s", rack)
	}
	if int64(s.cfg.Now()) >= c.Expires {
		return nil, errors.Wrap(access.ErrDenied, "grant expired")
	}
	return set.New(c.Secrets...), nil
}

// Resolve returns the plain text value of each stored secret with the given keys, for
// rack to connect with. Rack may read a secret when something on the rack uses it, or
// when grant names it. Grant may be empty. Resolve returns access.ErrDenied when rack
// may not read a secret, query.ErrNotFound when a secret does not exist, and a
// validation error for a file secret, whose value only the file holds.
func (s *Service) Resolve(
	ctx context.Context,
	tx gorp.Tx,
	rack ontology.ID,
	grant string,
	keys ...Key,
) (map[Key][]byte, error) {
	var secrets []Secret
	if err := s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&secrets).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	uses, err := s.cfg.Uses(ctx, gorp.OverrideTx(s.cfg.DB, tx), keys)
	if err != nil {
		return nil, err
	}
	readable := make(set.Set[Key], len(keys))
	for _, use := range uses {
		if slices.Contains(use.Racks, rack) {
			readable.Add(use.Secret)
		}
	}
	var granted set.Set[Key]
	values := make(map[Key][]byte, len(secrets))
	for _, sec := range secrets {
		base := sec.Base()
		if !readable.Contains(base.Key) {
			if granted == nil {
				if granted, err = s.granted(grant, rack); err != nil {
					return nil, err
				}
			}
			if !granted.Contains(base.Key) {
				return nil, errors.Wrapf(
					access.ErrDenied,
					"%s cannot read secret %q",
					rack,
					base.Name,
				)
			}
		}
		stored, ok := sec.Variant.(StoredSecret)
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"secret %q is a file, so only the file holds its value",
				base.Name,
			)
		}
		if values[base.Key], err = s.aead.Open(
			nil,
			nil,
			stored.Value,
			base.Key[:],
		); err != nil {
			return nil, errors.Wrapf(err, "decrypt secret %q", base.Name)
		}
	}
	return values, nil
}
