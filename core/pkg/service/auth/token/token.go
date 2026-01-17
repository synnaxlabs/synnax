// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package token

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

func isVerificationError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "verification")
}

func isExpiredError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "expired")
}

type ServiceConfig struct {
	// KeyProvider is the service used to generate and validate keys.
	// [REQUIRED]
	KeyProvider security.KeyProvider
	// Now is a function that returns the current time. This can be used to override the
	// current time.
	// [OPTIONAL] [DEFAULT: time.Now]
	Now func() time.Time
	// Expiration is the duration that the token will be valid for.
	// [OPTIONAL] [DEFAULT: 1 hour]
	Expiration time.Duration
	// RefreshThreshold is the duration before the token expires that a new token will be
	// issued.
	// [OPTIONAL] [DEFAULT: 5 minutes]
	RefreshThreshold time.Duration
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.KeyProvider = override.Nil(c.KeyProvider, other.KeyProvider)
	c.Expiration = override.Numeric(c.Expiration, other.Expiration)
	c.RefreshThreshold = override.Numeric(c.RefreshThreshold, other.RefreshThreshold)
	c.Now = override.Nil(c.Now, other.Now)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("auth")
	validate.NotNil(v, "key_provider", c.KeyProvider)
	validate.Positive(v, "expiration", c.Expiration)
	validate.Positive(v, "refresh_threshold", c.RefreshThreshold)
	validate.NotNil(v, "now", c.Now)
	return v.Error()
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{
		Expiration:       time.Hour,
		RefreshThreshold: time.Minute * 5,
		Now:              time.Now,
	}
)

// Service is a service for generating and validating tokens with UUID issuers.
type Service struct{ cfg ServiceConfig }

// NewService creates a new token service with the provided configuration. Returns a
// validate.Error if the configuration is invalid. See token.ServiceConfig for
// configuration details.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// New issues a new token for the given issuer. Returns the token as a string, and
// any errors encountered during signing.
func (s *Service) New(issuer uuid.UUID) (string, error) {
	method, key := s.signingMethodAndKey()
	now := s.cfg.Now().UTC()
	claims := jwt.NewWithClaims(method, jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    issuer.String(),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.Expiration)),
	})
	v, err := claims.SignedString(key)
	if err != nil {
		return v, auth.InvalidToken
	}
	return v, nil
}

// Validate validates the given token. Returns the UUID of the issuer along with any
// errors encountered.
func (s *Service) Validate(token string) (uuid.UUID, error) {
	id, _, err := s.validate(token)
	return id, err
}

// ValidateMaybeRefresh validates the given token. If the token is close to expiration
// (as defined by the RefreshThreshold), a new token will be issued and returned as well.
func (s *Service) ValidateMaybeRefresh(token string) (uuid.UUID, string, error) {
	id, claims, err := s.validate(token)
	if err != nil {
		return id, "", err
	}
	if s.isCloseToExpired(claims) {
		tk, err := s.New(id)
		return id, tk, err
	}
	return id, "", nil
}

func (s *Service) validate(token string) (uuid.UUID, *jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		return s.publicKey(), nil
	})
	if err != nil {
		if isVerificationError(err) {
			return uuid.Nil, claims, auth.InvalidToken
		}
		if isExpiredError(err) {
			return uuid.Nil, claims, auth.ExpiredToken
		}
		return uuid.Nil, claims, errors.Wrap(auth.Error, err.Error())
	}
	id, err := uuid.Parse(claims.Issuer)
	if err != nil {
		return uuid.Nil, claims, errors.Wrap(auth.Error, err.Error())
	}
	return id, claims, nil
}

func (s *Service) isCloseToExpired(claims *jwt.RegisteredClaims) bool {
	expiration := claims.ExpiresAt.UTC()
	currentTime := s.cfg.Now().UTC()
	if expiration.Sub(currentTime) < 0 {
		return false
	}
	return expiration.Sub(currentTime) < s.cfg.RefreshThreshold
}

func (s *Service) signingMethodAndKey() (jwt.SigningMethod, any) {
	key := s.cfg.KeyProvider.NodePrivate()
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return jwt.SigningMethodRS512, key
	case *ecdsa.PrivateKey:
		switch k.Curve.Params().BitSize {
		case 256:
			return jwt.SigningMethodES256, key
		case 384:
			return jwt.SigningMethodES384, key
		default:
			return jwt.SigningMethodES512, key
		}
	case *ed25519.PrivateKey:
		return jwt.SigningMethodEdDSA, key
	}
	panic("unsupported key type")
}

func (s *Service) publicKey() any {
	key := s.cfg.KeyProvider.NodePrivate()
	switch key := key.(type) {
	case *rsa.PrivateKey:
		return key.Public()
	case *ecdsa.PrivateKey:
		return key.Public()
	case *ed25519.PrivateKey:
		return key.Public()
	}
	panic("unsupported key type")
}
