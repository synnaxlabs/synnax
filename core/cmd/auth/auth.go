// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/cmd/access"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

func ProvisionRootUser(
	ctx context.Context,
	creds auth.InsecureCredentials,
	dist *distribution.Layer,
	svc *service.Layer,
) error {
	return dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var rootUser user.User
		if err := svc.User.NewRetrieve().
			WhereUsernames(creds.Username).
			Entry(&rootUser).
			Exec(ctx, tx); errors.Skip(err, query.NotFound) != nil {
			return err
		}
		if rootUser.Key == uuid.Nil {
			rootUser.Username = creds.Username
			rootUser.RootUser = true
			if err := svc.Auth.NewWriter(tx).Register(ctx, creds); err != nil {
				return err
			}
			if err := svc.User.NewWriter(tx).Create(ctx, &rootUser); err != nil {
				return err
			}
		}
		roles, err := access.Provision(ctx, tx, svc.RBAC)
		if err != nil {
			return err
		}
		// Migrate other users from legacy permissions
		return access.MigratePermissions(ctx, tx, dist, svc, roles)
	})
}
