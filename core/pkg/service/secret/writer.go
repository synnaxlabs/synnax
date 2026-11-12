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
	"strings"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, changes, and deletes secrets.
type Writer struct {
	tx         gorp.Tx
	aead       cipher.AEAD
	table      *gorp.Table[Key, Secret]
	now        func() telem.TimeStamp
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Set creates the secret, or replaces the secret with the same key. For a stored
// secret, value is the plain text, which Set encrypts and must not be empty. For a file
// secret, value must be empty. A replace must keep the source of the secret. Set gives
// a secret without a key a new one, and sets Updated. The commit carries a create
// action that holds the secret without its value.
func (w Writer) Set(ctx context.Context, s *Secret, value []byte) error {
	if err := validateSecret(*s, value); err != nil {
		return err
	}
	base := s.Base()
	if base.Key == uuid.Nil() {
		base.Key = uuid.New()
	} else if err := w.checkSource(ctx, base.Key, s.Source()); err != nil {
		return err
	}
	base.Updated = w.now()
	s.SetBase(base)
	if stored, ok := s.Variant.(StoredSecret); ok {
		stored.Value = w.aead.Seal(nil, nil, value, base.Key[:])
		s.Variant = stored
	}
	if err := w.table.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	created := *s
	if stored, ok := created.Variant.(StoredSecret); ok {
		stored.Value = nil
		created.Variant = stored
	}
	w.dispatcher.Attach(w.tx, base.Key, NewCreateAction(CreatePayload{Secret: created}))
	return nil
}

// checkSource returns a validation error when a secret with key exists and has a source
// other than source.
func (w Writer) checkSource(ctx context.Context, key Key, source Source) error {
	var prev []Secret
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Secret](key)).
		SkipMissingKeys().
		Entries(&prev).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	v := validate.New("secret")
	for _, p := range prev {
		v.Ternaryf(
			"source",
			p.Source() != source,
			"cannot change from %s to %s",
			p.Source(),
			source,
		)
	}
	return v.Error()
}

// Apply applies the actions to the secret with the given key and attaches them to the
// transaction of w, so its commit broadcasts them with the dispatch key of w. Apply
// returns an error wrapping validate.ErrValidation when an action is a create or the
// result fails to validate, and an error wrapping query.ErrNotFound when no secret has
// the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Secret](key)).
		ChangeErr(func(_ gorp.Context, s Secret) (Secret, error) {
			next, err := Reduce(s, acts...)
			if err != nil {
				return Secret{}, err
			}
			if err = next.Validate(); err != nil {
				return Secret{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// Rename renames the secret with the given key by applying a set of the name. It keeps
// the value.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.Apply(
		ctx,
		key,
		NewSetAction(SetPayload{Path: []string{"name"}, Value: name}),
	)
}

// Delete deletes the secrets with the given keys. It returns a gorp.RestrictError when
// anything uses one of them. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Secret](keys...)).
		Exec(ctx, w.tx)
}

// ValidateFileName adds an error to v at field unless name is a file name, so a rack
// that joins it to a folder stays in that folder. A colon starts a Windows drive, such
// as C:key.pem.
func ValidateFileName(v *validate.Validator, field, name string) {
	if name == "" {
		v.Ternary(field, true, "required")
		return
	}
	v.Ternary(
		field,
		name == "." || name == ".." || strings.ContainsAny(name, `/\:`),
		"must be a file name, not a path",
	)
}

func validateSecret(s Secret, value []byte) error {
	v := validate.New("secret")
	switch variant := s.Variant.(type) {
	case StoredSecret:
		v.Ternary("value", len(value) == 0, "required")
	case FileSecret:
		ValidateFileName(v, "file_name", variant.FileName)
		v.Ternary("value", len(value) != 0, "must be empty for a file secret")
	default:
		v.Ternary("source", true, "required")
	}
	v.Exec(s.Validate)
	return v.Error()
}
