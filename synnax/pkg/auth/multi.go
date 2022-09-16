package auth

import (
	"github.com/synnaxlabs/synnax/pkg/auth/password"
	"github.com/synnaxlabs/x/gorp"
)

// MultiAuthenticator implements the Authenticator interface by wrapping a set of
// exiting Authenticator(s). This is useful for combining multiple Authentication
// sources into a single interface. Authenticator(s) are executed in order,
// and the first Authenticator to succeeds (i.e. non-nil error) is used for the operation.
type MultiAuthenticator []Authenticator

var _ Authenticator = MultiAuthenticator{}

// Authenticate implements the Authenticator interface.
func (a MultiAuthenticator) Authenticate(creds InsecureCredentials) error {
	var err error
	for _, auth := range a {
		if err = auth.Authenticate(creds); err == nil {
			return nil
		}
	}
	return err
}

// NewWriter implements the Authenticator interface.
func (a MultiAuthenticator) NewWriter() Writer {
	var w multiWriter
	for _, auth := range a {
		w = append(w, auth.NewWriter())
	}
	return w
}

// NewWriterUsingTxn implements the Authenticator interface.
func (a MultiAuthenticator) NewWriterUsingTxn(txn gorp.Txn) Writer {
	var w multiWriter
	for _, auth := range a {
		w = append(w, auth.NewWriterUsingTxn(txn))
	}
	return w
}

type multiWriter []Writer

// Register implements the Authenticator interface.
func (w multiWriter) Register(creds InsecureCredentials) error {
	var err error
	for _, auth := range w {
		if err = auth.Register(creds); err == nil {
			return nil
		}
	}
	return err
}

// UpdateUsername implements the Authenticator interface.
func (w multiWriter) UpdateUsername(creds InsecureCredentials, newUser string) error {
	var err error
	for _, auth := range w {
		if err = auth.UpdateUsername(creds, newUser); err == nil {
			return nil
		}
	}
	return err
}

// UpdatePassword implements the Authenticator interface.
func (w multiWriter) UpdatePassword(creds InsecureCredentials, newPass password.Raw) error {
	var err error
	for _, auth := range w {
		if err = auth.UpdatePassword(creds, newPass); err == nil {
			return nil
		}
	}
	return err
}
