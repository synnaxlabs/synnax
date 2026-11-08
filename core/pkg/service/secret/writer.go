// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"context"
	"crypto/cipher"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, renames, and deletes secrets.
type Writer struct {
	tx    gorp.Tx
	aead  cipher.AEAD
	table *gorp.Table[Key, Secret]
	now   func() telem.TimeStamp
}

// Create creates the secret, or replaces the secret with the same key. For a stored
// secret, value is the plain text, which Create encrypts and must not be empty. For a
// file secret, value must be empty. Create gives a secret without a key a new one, and
// sets Updated.
func (w Writer) Create(ctx context.Context, s *Secret, value []byte) error {
	if err := validateSecret(*s, value); err != nil {
		return err
	}
	base := s.Base()
	if base.Key == uuid.Nil() {
		base.Key = uuid.New()
	}
	base.Updated = w.now()
	s.SetBase(base)
	if stored, ok := s.Variant.(StoredSecret); ok {
		stored.Value = w.aead.Seal(nil, nil, value, base.Key[:])
		s.Variant = stored
	}
	return w.table.NewCreate().Entry(s).Exec(ctx, w.tx)
}

// Rename renames the secret with the given key, keeping its value. It returns
// query.ErrNotFound if no secret has the key, and a validation error if name is empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Secret](key)).
		ChangeErr(func(_ gorp.Context, s Secret) (Secret, error) {
			base := s.Base()
			base.Name = name
			s.SetBase(base)
			if err := s.Validate(); err != nil {
				return Secret{}, err
			}
			return s, nil
		}).
		Exec(ctx, w.tx)
}

// Delete deletes the secrets with the given keys. It returns a gorp.RestrictError when
// anything uses one of them. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Secret](keys...)).
		Exec(ctx, w.tx)
}

func validateSecret(s Secret, value []byte) error {
	v := validate.New("secret")
	switch variant := s.Variant.(type) {
	case StoredSecret:
		v.Ternary("value", len(value) == 0, "required")
	case FileSecret:
		v.NotEmptyString("path", variant.Path)
		v.Ternary("value", len(value) != 0, "must be empty for a file secret")
	default:
		v.Ternary("source", true, "required")
	}
	v.Exec(s.Validate)
	return v.Error()
}
