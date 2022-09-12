package api

import (
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/password"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/arya-analytics/x/gorp"
	roacherrors "github.com/cockroachdb/errors"
)

// AuthService is the core authentication service for the delta API.
type AuthService struct {
	LoggingProvider
	ValidationProvider
	DBProvider
	AuthProvider
	UserProvider
}

func NewAuthService(p Provider) *AuthService {
	return &AuthService{
		LoggingProvider:    p.Logging,
		ValidationProvider: p.Validation,
		DBProvider:         p.DB,
		AuthProvider:       p.Auth,
		UserProvider:       p.User,
	}
}

// Login attempts to authenticate a user with the provided credentials. If successful,
// returns a response containing a valid JWT along with the user's details.
func (s *AuthService) Login(creds auth.InsecureCredentials) (tr TokenResponse, _ errors.Typed) {
	if err := s.Validate(&creds); err.Occurred() {
		return tr, err
	}
	if err := s.Authenticator.Authenticate(creds); err != nil {
		if roacherrors.Is(err, auth.InvalidCredentials) {
			return tr, errors.Auth(err)
		}
		return tr, errors.Unexpected(err)
	}
	u, err := s.User.RetrieveByUsername(creds.Username)
	if err != nil {
		return tr, errors.Unexpected(err)
	}
	return s.tokenResponse(u)
}

// RegistrationRequest is an API request to register a new user.
type RegistrationRequest struct {
	auth.InsecureCredentials
}

// Register registers new user with the provided credentials. If successful, returns a
// response containing a valid JWT along with the user's details.
func (s *AuthService) Register(req RegistrationRequest) (tr TokenResponse, _ errors.Typed) {
	if err := s.Validate(req); err.Occurred() {
		return tr, err
	}
	return tr, s.WithTxn(func(txn gorp.Txn) errors.Typed {
		aw := s.Authenticator.NewWriterUsingTxn(txn)
		if err := aw.Register(req.InsecureCredentials); err != nil {
			return errors.General(err)
		}
		u := &user.User{Username: req.Username}
		userWriter := s.User.NewWriterUsingTxn(txn)
		if err := userWriter.Create(u); err != nil {
			return errors.General(err)
		}
		var tErr errors.Typed
		tr, tErr = s.tokenResponse(*u)
		return tErr
	})
}

// ChangePasswordRequest is an API request to change the password for a user.
type ChangePasswordRequest struct {
	auth.InsecureCredentials
	NewPassword password.Raw `json:"new_password" msgpack:"new_password" validate:"required"`
}

// ChangePassword changes the password for the user with the provided credentials.
func (s *AuthService) ChangePassword(cpr ChangePasswordRequest) errors.Typed {
	if err := s.Validate(cpr); err.Occurred() {
		return err
	}
	return s.WithTxn(func(txn gorp.Txn) errors.Typed {
		return errors.MaybeGeneral(s.Authenticator.NewWriterUsingTxn(txn).
			UpdatePassword(cpr.InsecureCredentials, cpr.NewPassword))
	})
}

// ChangeUsernameRequest is an API request to change the username for a user.
type ChangeUsernameRequest struct {
	auth.InsecureCredentials `json:"" msgpack:""`
	NewUsername              string `json:"new_username" msgpack:"new_username" validate:"required"`
}

// ChangeUsername changes the username for the user with the provided credentials.
func (s *AuthService) ChangeUsername(cur ChangeUsernameRequest) errors.Typed {
	if err := s.Validate(&cur); err.Occurred() {
		return err
	}
	return s.WithTxn(func(txn gorp.Txn) errors.Typed {
		authWriter := s.Authenticator.NewWriterUsingTxn(txn)
		if err := authWriter.UpdateUsername(
			cur.InsecureCredentials,
			cur.NewUsername,
		); err != nil {
			return errors.Unexpected(err)
		}
		u, err := s.User.RetrieveByUsername(cur.NewUsername)
		if err != nil {
			return errors.General(err)
		}
		u.Username = cur.InsecureCredentials.Username
		return errors.MaybeUnexpected(s.User.NewWriterUsingTxn(txn).Update(u))
	})
}

// TokenResponse is a response containing a valid JWT along with details about the user
// the token is associated with.
type TokenResponse struct {
	// User is the user the token is associated with.
	User user.User `json:"user"`
	// Token is the JWT.
	Token string `json:"token"`
}

func (s *AuthService) tokenResponse(u user.User) (TokenResponse, errors.Typed) {
	tk, err := s.Token.New(u.Key)
	return TokenResponse{User: u, Token: tk}, errors.MaybeUnexpected(err)
}
