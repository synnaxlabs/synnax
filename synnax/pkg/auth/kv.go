// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"
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
func (db *KV) Authenticate(ctx context.Context, creds InsecureCredentials) error {
	_, err := db.authenticate(ctx, creds)
	return err
}

func (db *KV) authenticate(ctx context.Context, creds InsecureCredentials) (SecureCredentials, error) {
	secureCreds, err := db.retrieve(ctx, db.DB, creds.Username)
	if err != nil {
		if err == query.NotFound {
			return secureCreds, InvalidCredentials
		}
		return secureCreds, err
	}
	return secureCreds, secureCreds.Password.Validate(creds.Password)
}

// NewWriter implements the Authenticator interface.
func (db *KV) NewWriter() Writer { return db.NewWriterWithTxn(db.DB) }

// NewWriterWithTxn implements the Authenticator interface.
func (db *KV) NewWriterWithTxn(txn gorp.TypedWriter) Writer { return &kvWriter{kv: db, txn: txn} }

func (db *KV) exists(ctx context.Context, txn gorp.TypedWriter, user string) (bool, error) {
	return gorp.NewRetrieve[string, SecureCredentials]().
		WhereKeys(user).
		Exists(ctx, txn)
}

func (db *KV) retrieve(ctx context.Context, txn gorp.TypedWriter, user string) (SecureCredentials, error) {
	var creds SecureCredentials
	return creds, gorp.NewRetrieve[string, SecureCredentials]().
		WhereKeys(user).
		Entry(&creds).
		Exec(ctx, txn)
}

type kvWriter struct {
	kv  *KV
	txn gorp.TypedWriter
}

// Register implements the sec.authenticator interface.
func (w *kvWriter) Register(ctx context.Context, creds InsecureCredentials) error {
	err := w.checkUsernameExists(ctx, creds.Username)
	if err != nil {
		return err
	}
	sec := SecureCredentials{Username: creds.Username}
	sec.Password, err = creds.Password.Hash()
	if err != nil {
		return err
	}
	return w.set(ctx, sec)
}

// UpdateUsername implements the sec.authenticator interface.
func (w *kvWriter) UpdateUsername(ctx context.Context, creds InsecureCredentials, newUser string) error {
	secureCreds, err := w.kv.authenticate(ctx, creds)
	if err != nil {
		return err
	}
	if err = w.checkUsernameExists(ctx, newUser); err != nil {
		return err
	}
	secureCreds.Username = newUser
	return w.set(ctx, secureCreds)
}

func (w *kvWriter) checkUsernameExists(ctx context.Context, user string) error {
	exists, err := w.kv.exists(ctx, w.txn, user)
	if exists {
		return errors.New("[auth] - username already registered")
	}
	return err
}

// UpdatePassword implements the sec.authenticator interface.
func (w *kvWriter) UpdatePassword(ctx context.Context, creds InsecureCredentials, newPass password.Raw) error {
	secureCreds, err := w.kv.authenticate(ctx, creds)
	if err != nil {
		return err
	}
	secureCreds.Password, err = newPass.Hash()
	if err != nil {
		return err
	}
	return w.set(ctx, secureCreds)
}

func (w *kvWriter) set(ctx context.Context, creds SecureCredentials) error {
	return gorp.NewCreate[string, SecureCredentials]().Entry(&creds).Exec(ctx, w.txn)
}
