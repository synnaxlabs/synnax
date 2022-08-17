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
