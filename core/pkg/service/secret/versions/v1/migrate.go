// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v3"
	v0 "github.com/synnaxlabs/synnax/pkg/service/secret/versions/v0"
)

// NewMigration returns the migration that lifts stored secrets from v0 to v1. Each v0
// file secret moves to rk.
func NewMigration(rk rack.Key) gorp.Migration {
	return migrate.NewEntry(
		"v63_file_rack",
		func(ctx context.Context, old v0.Secret) (Secret, error) {
			s, err := autoMigrateSecret(ctx, old)
			if err != nil {
				return Secret{}, err
			}
			if file, ok := s.Variant.(FileSecret); ok {
				file.Rack = rk
				s.Variant = file
			}
			return s, nil
		},
	)
}
