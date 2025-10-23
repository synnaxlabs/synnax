// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/validate"
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
		} else if !rootUser.RootUser {
			return errors.Wrapf(
				validate.Error,
				"a user with username %s exists but is not a root user",
				rootUser.Username,
			)
		}
		roleKey, err := access.ProvisionRootRole(ctx, tx, svc.RBAC)
		if err != nil {
			return err
		}
		return svc.RBAC.Role.NewWriter(tx).AssignRole(ctx, user.OntologyID(rootUser.Key), roleKey)
	})
}
