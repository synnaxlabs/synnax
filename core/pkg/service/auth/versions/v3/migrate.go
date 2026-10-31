// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v2 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// NewMigration returns the migration that stamps each stored session with the current
// version of its credential, which version reads within the migration's transaction.
// It deletes each session whose credential is gone.
func NewMigration(
	version func(context.Context, gorp.Tx, uuid.UUID) (telem.TimeStamp, error),
) gorp.Migration {
	return migrate.New(
		"v3_session_credential_version",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var old []v2.Session
			if err := gorp.NewRetrieve[v2.SessionKey, v2.Session]().
				Entries(&old).
				Exec(ctx, tx); err != nil {
				return err
			}
			var (
				sessions = make([]Session, 0, len(old))
				gone     []SessionKey
			)
			for _, o := range old {
				v, err := version(ctx, tx, o.Credential)
				if errors.Is(err, query.ErrNotFound) {
					gone = append(gone, o.Key)
					continue
				}
				if err != nil {
					return err
				}
				s, err := autoMigrateSession(ctx, o)
				if err != nil {
					return err
				}
				s.CredentialVersion = v
				sessions = append(sessions, s)
			}
			if err := gorp.WrapWriter[SessionKey, Session](tx).
				Set(ctx, sessions...); err != nil {
				return err
			}
			return gorp.WrapWriter[SessionKey, Session](tx).Delete(ctx, gone...)
		},
	)
}
