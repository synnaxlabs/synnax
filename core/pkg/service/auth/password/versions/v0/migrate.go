// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"context"
	"crypto/sha1"
	"uuid"

	"github.com/synnaxlabs/alamos"
	authv1 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	userv0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// keyNamespace salts the keys the move derives, so they cannot collide with a key
// derived from the same subject by anything else.
var keyNamespace = uuid.MustParse("3f3a1b0e-7c2e-4a8f-9d6b-2a1c5e8f0b41")

// migratedKey derives the password key from its subject as a name-based UUID (RFC
// 9562, version 5). Every node of a Core that runs the move writes the same record,
// so two nodes migrating before replication meets cannot leave two passwords behind.
func migratedKey(subject ontology.ID) Key {
	h := sha1.New()
	h.Write(keyNamespace[:])
	h.Write([]byte(subject.String()))
	var key Key
	copy(key[:], h.Sum(nil))
	key[6] = (key[6] & 0x0f) | 0x50
	key[8] = (key[8] & 0x3f) | 0x80
	return key
}

// NewMove returns the migration that moves every row of the credentials table into
// the password table, matching each username to a user in the user table. The hash
// moves as it is, still tagged bcrypt. A credential whose username no user holds is
// dropped with a warning. Each password gets a key derived from its subject, so the
// move is the same on every node. It reads users in their v0 shape, so it must run
// after the user table's migrations.
func NewMove() migrate.Migration {
	return gorp.NewMigration("v0.move_credentials_to_passwords", move)
}

// subjectsByUsername maps every stored user's username to its ontology ID.
func subjectsByUsername(
	ctx context.Context,
	tx gorp.Tx,
) (_ map[string]ontology.ID, err error) {
	iter, err := gorp.WrapReader[userv0.Key, userv0.User](tx).
		OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	subjects := make(map[string]ontology.ID)
	for iter.First(); iter.Valid(); iter.Next() {
		u := iter.Value(ctx)
		if err = iter.Error(); err != nil {
			return nil, err
		}
		subjects[u.Username] = u.OntologyID()
	}
	return subjects, nil
}

func move(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) (err error) {
	subjects, err := subjectsByUsername(ctx, tx)
	if err != nil {
		return err
	}
	reader := gorp.WrapReader[string, authv1.SecureCredentials](tx)
	iter, err := reader.OpenIterator(gorp.IterOptions{})
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	var (
		passwords []Password
		usernames []string
	)
	for iter.First(); iter.Valid(); iter.Next() {
		old := iter.Value(ctx)
		if err = iter.Error(); err != nil {
			return err
		}
		usernames = append(usernames, old.Username)
		subject, ok := subjects[old.Username]
		if !ok {
			ins.L.Warn(
				"dropping stored credential with no user",
				zap.String("username", old.Username),
			)
			continue
		}
		passwords = append(passwords, Password{
			Key:      migratedKey(subject),
			Subject:  subject,
			Username: old.Username,
			Hash:     old.Password,
			Created:  telem.Now(),
		})
	}
	if len(usernames) == 0 {
		return nil
	}
	if err = gorp.WrapWriter[Key, Password](tx).Set(ctx, passwords...); err != nil {
		return err
	}
	return gorp.WrapWriter[string, authv1.SecureCredentials](tx).
		Delete(ctx, usernames...)
}
