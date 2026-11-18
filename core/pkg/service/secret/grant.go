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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Grant creates a grant that lets subject run cmd once with the secrets with the given
// keys, until the grant expires. It makes no access check and does not check that the
// task of cmd is on the subject rack: the caller does both. It returns
// query.ErrNotFound when a secret does not exist, and a validation error when subject
// is not a rack or keys is empty.
func (s *Service) Grant(
	ctx context.Context,
	tx gorp.Tx,
	subject resource.ID,
	cmd task.Command,
	keys ...Key,
) (grant.Key, error) {
	if err := s.NewRetrieve().Where(MatchKeys(keys...)).Exec(ctx, tx); err != nil {
		return uuid.Nil(), err
	}
	g := grant.Grant{Subject: subject, Secrets: keys, Command: cmd}
	if err := s.cfg.Grant.NewWriter(tx).Create(ctx, &g); err != nil {
		return uuid.Nil(), err
	}
	return g.Key, nil
}

// Resolve spends the grant with the given key and returns its command, with the plain
// text value of each secret that the grant names, for reader to run the command on the
// task with taskKey. The Core's integrations call it in process for the Core's rack.
// Resolve returns access.ErrDenied when the grant has ended, is for another subject,
// or is for another task. A grant for another task is spent. It returns
// query.ErrNotFound when a secret no longer exists. It reads a file secret only for
// the Core's rack, as Reveal does, and returns a validation error for one that another
// rack reads itself. Resolve opens its own snapshot and transaction, so the caller must
// not hold either.
func (s *Service) Resolve(
	ctx context.Context,
	reader resource.ID,
	taskKey task.Key,
	grantKey grant.Key,
) (task.Command, map[Key][]byte, error) {
	g, err := s.cfg.Grant.Redeem(ctx, grantKey, reader, taskKey)
	if err != nil {
		return task.Command{}, nil, err
	}
	var values map[Key][]byte
	if err = s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
		values, err = s.reveal(ctx, snap, reader == s.cfg.Rack.ResourceID(), g.Secrets)
		return err
	}); err != nil {
		return task.Command{}, nil, err
	}
	return g.Command, values, nil
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
	return s.reveal(ctx, snap, true, keys)
}

// reveal returns the values of the secrets with keys. It reads file secrets only when
// files is true.
func (s *Service) reveal(
	ctx context.Context,
	snap gorp.Snapshot,
	files bool,
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
		value, err := s.open(sec, files)
		if err != nil {
			return nil, err
		}
		values[sec.Base().Key] = value
	}
	return values, nil
}

// open returns the value of sec. The value of a file secret is the file without the
// "\r" and "\n" characters at its end, so a file that ends with a line ending works.
func (s *Service) open(sec Secret, files bool) ([]byte, error) {
	base := sec.Base()
	switch variant := sec.Variant.(type) {
	case StoredSecret:
		value, err := s.aead.Open(nil, nil, variant.Value, base.Key[:])
		if err != nil {
			return nil, errors.Wrapf(err, "decrypt secret %q", base.Name)
		}
		return value, nil
	case FileSecret:
		if !files || variant.Rack != s.cfg.Rack {
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
