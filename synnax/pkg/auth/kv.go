package auth

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/synnax/pkg/auth/password"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// KV is a simple key-value backed Authenticator. It saves data to the provided
// gorp DB. It's important to note that all gorp.txn(s) provided to the Authenticator
// interface must be spawned from the same gorp DB.
type KV struct{ DB *gorp.DB }

var _ Authenticator = (*KV)(nil)
var _ Authenticator = (*KV)(nil)

// Authenticate implements the Authenticator interface.
func (db *KV) Authenticate(creds InsecureCredentials) error {
	_, err := db.authenticate(creds)
	return err
}

func (db *KV) authenticate(creds InsecureCredentials) (SecureCredentials, error) {
	secureCreds, err := db.retrieve(db.DB, creds.Username)
	if err != nil {
		if err == query.NotFound {
			return secureCreds, InvalidCredentials
		}
		return secureCreds, err
	}
	return secureCreds, secureCreds.Password.Validate(creds.Password)
}

// NewWriter implements the Authenticator interface.
func (db *KV) NewWriter() Writer { return db.NewWriterUsingTxn(db.DB) }

// NewWriterUsingTxn implements the Authenticator interface.
func (db *KV) NewWriterUsingTxn(txn gorp.Txn) Writer { return &kvWriter{kv: db, txn: txn} }

func (db *KV) exists(txn gorp.Txn, user string) (bool, error) {
	return gorp.NewRetrieve[string, SecureCredentials]().
		WhereKeys(user).
		Exists(txn)
}

func (db *KV) retrieve(txn gorp.Txn, user string) (SecureCredentials, error) {
	var creds SecureCredentials
	return creds, gorp.NewRetrieve[string, SecureCredentials]().
		WhereKeys(user).
		Entry(&creds).
		Exec(txn)
}

type kvWriter struct {
	kv  *KV
	txn gorp.Txn
}

// Register implements the sec.authenticator interface.
func (w *kvWriter) Register(creds InsecureCredentials) error {
	err := w.checkUsernameExists(creds.Username)
	if err != nil {
		return err
	}
	sec := SecureCredentials{Username: creds.Username}
	sec.Password, err = creds.Password.Hash()
	if err != nil {
		return err
	}
	return w.set(sec)
}

// UpdateUsername implements the sec.authenticator interface.
func (w *kvWriter) UpdateUsername(creds InsecureCredentials, newUser string) error {
	secureCreds, err := w.kv.authenticate(creds)
	if err != nil {
		return err
	}
	if err = w.checkUsernameExists(newUser); err != nil {
		return err
	}
	secureCreds.Username = newUser
	return w.set(secureCreds)
}

func (w *kvWriter) checkUsernameExists(user string) error {
	exists, err := w.kv.exists(w.txn, user)
	if exists {
		return errors.New("[auth] - username already registered")
	}
	return err
}

// UpdatePassword implements the sec.authenticator interface.
func (w *kvWriter) UpdatePassword(creds InsecureCredentials, newPass password.Raw) error {
	secureCreds, err := w.kv.authenticate(creds)
	if err != nil {
		return err
	}
	secureCreds.Password, err = newPass.Hash()
	if err != nil {
		return err
	}
	return w.set(secureCreds)
}

func (w *kvWriter) set(creds SecureCredentials) error {
	return gorp.NewCreate[string, SecureCredentials]().Entry(&creds).Exec(w.txn)
}
