package token

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/security"
	"time"
)

// Service is a service for generating and validating tokens with UUID issuers.
type Service struct {
	// KeyService is the service used to generate and validate keys.
	KeyService security.KeyProvider
	// Expiration is the duration that the token will be valid for.
	Expiration time.Duration
	// RefreshThreshold is the duration before the token expires that a new token will be
	// issued.
	RefreshThreshold time.Duration
}

// New issues a new token for the given issuer. Returns the token as a string, and
// any errors encountered during signing.
func (s *Service) New(issuer uuid.UUID) (string, error) {
	method, key := s.signingMethodAndKey()
	claims := jwt.NewWithClaims(method, jwt.StandardClaims{
		IssuedAt:  time.Now().Unix(),
		Issuer:    issuer.String(),
		ExpiresAt: time.Now().Add(s.Expiration).Unix(),
	})
	return claims.SignedString(key)
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

func (s *Service) validate(token string) (uuid.UUID, *jwt.StandardClaims, error) {
	claims := &jwt.StandardClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return s.publicKey(), nil
	})
	if err != nil {
		return uuid.Nil, claims, err
	}
	id, err := uuid.Parse(claims.Issuer)
	return id, claims, err
}

func (s *Service) isCloseToExpired(claims *jwt.StandardClaims) bool {
	return time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) < s.RefreshThreshold
}

func (s *Service) signingMethodAndKey() (jwt.SigningMethod, interface{}) {
	key := s.KeyService.NodeSecret()
	switch key.(type) {
	case *rsa.PrivateKey:
		return jwt.SigningMethodRS512, key
	case *ecdsa.PrivateKey:
		return jwt.SigningMethodES512, key
	case *ed25519.PrivateKey:
		return jwt.SigningMethodEdDSA, key
	}
	panic("unsupported key type")
}

func (s *Service) publicKey() interface{} {
	key := s.KeyService.NodeSecret()
	switch key.(type) {
	case *rsa.PrivateKey:
		return key.(*rsa.PrivateKey).Public()
	case *ecdsa.PrivateKey:
		return key.(*ecdsa.PrivateKey).Public()
	case *ed25519.PrivateKey:
		return key.(*ed25519.PrivateKey).Public()
	}
	panic("unsupported key type")
}
