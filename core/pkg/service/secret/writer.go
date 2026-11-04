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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, renames, and deletes secrets.
type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	aead      cipher.AEAD
	table     *gorp.Table[Key, Secret]
	uses      Finder
	now       func() telem.TimeStamp
	group     group.Group
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
	if err := w.table.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := s.OntologyID()
	if err := w.otgWriter.DefineResources(ctx, otgID); err != nil {
		return err
	}
	return w.otgWriter.DefineRelationships(
		ctx,
		w.group.OntologyID(),
		ontology.RelationshipTypeParentOf,
		otgID,
	)
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

// Delete deletes the secrets with the given keys. It returns a validation error naming
// the resource that uses a secret when anything uses one. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	uses, err := w.uses(ctx, w.tx, keys)
	if err != nil {
		return err
	}
	if len(uses) > 0 {
		use := uses[0]
		var s Secret
		if err = w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Secret](use.Secret)).
			Entry(&s).
			Exec(ctx, w.tx); err != nil {
			return err
		}
		return errors.Wrapf(
			validate.ErrValidation,
			"cannot delete secret %q because %s %q uses it",
			s.Base().Name,
			use.Resource.Type,
			use.Name,
		)
	}
	if err = w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Secret](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...)
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
