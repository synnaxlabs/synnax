package auth

import (
	"github.com/arya-analytics/delta/pkg/auth/password"
)

var (
	// InvalidCredentials is returned when the credentials for a particular entity
	// are invalid.
	InvalidCredentials = password.Invalid
)
