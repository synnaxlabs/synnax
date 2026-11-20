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
	"bytes"
	"context"
	"io/fs"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Grant creates g, which lets its subject run its command once with the secrets and
// certificates that it names, until it expires. It makes no access check, does not
// check the certificates, and does not check that the task of the command is on the
// subject rack: the caller does all three. It returns query.ErrNotFound when a secret
// does not exist, and a validation error when the subject is not a rack, g names no
// secret and no certificate, or a secret is a file secret of another rack.
func (s *Service) Grant(
	ctx context.Context,
	tx gorp.Tx,
	g grant.Grant,
) (grant.Key, error) {
	if err := s.checkGranted(ctx, tx, g); err != nil {
		return uuid.Nil(), err
	}
	if err := s.cfg.Grant.NewWriter(tx).Create(ctx, &g); err != nil {
		return uuid.Nil(), err
	}
	return g.Key, nil
}

// checkGranted returns an error when a secret of g does not exist, or is a file secret
// of a rack other than the subject of g.
func (s *Service) checkGranted(ctx context.Context, tx gorp.Tx, g grant.Grant) error {
	if len(g.Secrets) == 0 {
		return nil
	}
	var secrets []Secret
	if err := s.NewRetrieve().
		Where(MatchKeys(g.Secrets...)).
		Entries(&secrets).
		Exec(ctx, tx); err != nil {
		return err
	}
	for _, sec := range secrets {
		file, ok := sec.Variant.(FileSecret)
		if ok && file.Rack.ResourceID() != g.Subject {
			return errors.Wrapf(
				validate.ErrValidation,
				"secret %q is a file of rack %d, not of %s",
				file.Name,
				file.Rack,
				g.Subject,
			)
		}
	}
	return nil
}

// Resolve spends the grant with the given key for reader to run its command on the
// task with taskKey. It returns the grant, which holds the command and the keys of the
// secrets it names, and the plain text values of those secrets as RevealTo reads them.
// The Core's integrations call it in process for the Core's rack. Resolve returns
// access.ErrDenied when the grant has ended, is for another subject, or is for another
// task. A grant for another task is spent. It returns query.ErrNotFound when a secret
// no longer exists. Resolve opens its own snapshot and transaction, so the caller must
// not hold either.
func (s *Service) Resolve(
	ctx context.Context,
	reader resource.ID,
	taskKey task.Key,
	grantKey grant.Key,
) (grant.Grant, map[Key][]byte, error) {
	g, err := s.cfg.Grant.Redeem(ctx, grantKey, reader, taskKey)
	if err != nil {
		return grant.Grant{}, nil, err
	}
	var values map[Key][]byte
	if err = s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
		values, err = s.RevealTo(ctx, snap, reader, g.Secrets...)
		return err
	}); err != nil {
		return grant.Grant{}, nil, err
	}
	return g, values, nil
}

// Reveal returns the plain text value of each secret with the given keys, with no
// access check. It is for Core code that connects with a secret itself, which must
// never send the values to a client. Reveal reads a file secret of the Core's rack
// from the Core's secrets folder. It returns query.ErrNotFound when a secret does not
// exist, and a validation error for a file secret of another rack, or when the Core
// has no secrets folder.
func (s *Service) Reveal(ctx context.Context, snap gorp.Snapshot, keys ...Key) (
	map[Key][]byte,
	error,
) {
	return s.reveal(ctx, snap, s.cfg.Rack, keys)
}

// RevealTo returns the plain text value of each secret with the given keys as reader
// reads them, with no access check. The Core's rack reads a file secret of its own
// from the Core's secrets folder. Another rack reads a file secret of its own from its
// own folder, so RevealTo returns no value for it. A file secret of a rack other than
// reader is a validation error. It returns query.ErrNotFound when a secret does not
// exist. reader must be a rack.
func (s *Service) RevealTo(
	ctx context.Context,
	snap gorp.Snapshot,
	reader resource.ID,
	keys ...Key,
) (map[Key][]byte, error) {
	rk, err := rack.KeyFromResourceID(reader)
	if err != nil {
		return nil, err
	}
	return s.reveal(ctx, snap, rk, keys)
}

// reveal returns the values of the secrets with keys as reader reads them.
func (s *Service) reveal(
	ctx context.Context,
	snap gorp.Snapshot,
	reader rack.Key,
	keys []Key,
) (map[Key][]byte, error) {
	var secrets []Secret
	if err := s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&secrets).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	values := make(map[Key][]byte, len(secrets))
	for _, sec := range secrets {
		file, ok := sec.Variant.(FileSecret)
		if ok && file.Rack == reader && reader != s.cfg.Rack {
			continue
		}
		value, err := s.open(sec, reader)
		if err != nil {
			return nil, err
		}
		values[sec.Base().Key] = value
	}
	return values, nil
}

// open returns the value of sec. The value of a file secret is the file without the
// "\r" and "\n" characters at its end, so a file that ends with a line ending works.
func (s *Service) open(sec Secret, reader rack.Key) ([]byte, error) {
	base := sec.Base()
	switch variant := sec.Variant.(type) {
	case StoredSecret:
		value, err := s.aead.Open(nil, nil, variant.Value, base.Key[:])
		if err != nil {
			return nil, errors.Wrapf(err, "decrypt secret %q", base.Name)
		}
		return value, nil
	case FileSecret:
		if variant.Rack != reader {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"secret %q is a file, so rack %d reads it from its own secrets folder",
				base.Name,
				variant.Rack,
			)
		}
		if s.cfg.Folder == nil {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"secret %q is a file, but the Core has no secrets folder",
				base.Name,
			)
		}
		value, err := fs.ReadFile(s.cfg.Folder, variant.FileName)
		if err != nil {
			return nil, errors.Wrapf(err, "read secret %q", base.Name)
		}
		return bytes.TrimRight(value, "\r\n"), nil
	default:
		return nil, errors.Newf("secret %q has no source", base.Name)
	}
}
