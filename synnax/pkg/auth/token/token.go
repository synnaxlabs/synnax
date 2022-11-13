package token

import (
	"crypto/rsa"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"time"
)

// Service is a service for generating and validating tokens with UUID issuers.
type Service struct {
	// Secret is the secret used to sign the token.
	Secret *rsa.PrivateKey
	// Expiration is the duration that the token will be valid for.
	Expiration time.Duration
	// RefreshThreshold is the duration before the token expires that a new token will be
	// issued.
	RefreshThreshold time.Duration
}

// New issues a new token for the given issuer. Returns the token as a string, and
// any errors encountered during signing.
func (s *Service) New(issuer uuid.UUID) (string, error) {
	claims := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.StandardClaims{
		IssuedAt:  time.Now().Unix(),
		Issuer:    issuer.String(),
		ExpiresAt: time.Now().Add(s.Expiration).Unix(),
	})
	return claims.SignedString(s.Secret)
}

// Validate validates the given token. Returns the UUID of the issuer along with any
// errors encountered.
func (s *Service) Validate(token string) (uuid.UUID, error) {
	claims := &jwt.StandardClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return s.Secret.Public(), nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(claims.Issuer)
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
		return s.Secret.Public(), nil
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
