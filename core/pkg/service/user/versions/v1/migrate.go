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
	"strings"

	"github.com/synnaxlabs/gorp/migrate"
	v0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
)

// Migration joins each user's first and last name into its name, falling back to the
// username when both are empty, then to "Unnamed user". The username stays only on the
// user's password.
var Migration = migrate.NewEntry(
	"user_name",
	func(ctx context.Context, old v0.User) (User, error) {
		u, err := autoMigrateUser(ctx, old)
		if err != nil {
			return User{}, err
		}
		u.Name = strings.TrimSpace(old.FirstName + " " + old.LastName)
		if u.Name == "" {
			u.Name = old.Username
		}
		if u.Name == "" {
			u.Name = "Unnamed user"
		}
		return u, nil
	},
)
