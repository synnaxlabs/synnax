package auth

import (
	"github.com/synnaxlabs/synnax/pkg/auth/password"
)

var (
	// InvalidCredentials is returned when the credentials for a particular entity
	// are invalid.
	InvalidCredentials = password.Invalid
)
