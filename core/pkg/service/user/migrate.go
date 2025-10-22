// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
)

var (
	rbacMigrationKey   = []byte("sy_user_rbac_migration_performed")
	rbacMigrationValue = []byte{1}
)

// MigrateToRBAC assigns the Administrator role to all existing root users.
// This is idempotent and safe to run multiple times.
func (s *Service) MigrateToRBAC(ctx context.Context, rbacService *rbac.Service) error {
	return s.DB.WithTx(ctx, func(tx gorp.Tx) error {
		// Check if migration has already been performed
		performed, closer, err := tx.Get(ctx, rbacMigrationKey)
		if err != nil && !errors.Is(err, kv.NotFound) {
			return err
		} else if err == nil {
			if err := closer.Close(); err != nil {
				return err
			}
		}
		if string(performed) == string(rbacMigrationValue) {
			// Migration already performed
			return nil
		}

		// Get the Administrator role
		var adminRoles []rbac.Role
		if err := rbacService.NewRoleRetriever().
			WhereName("Administrator").
			Entries(&adminRoles).
			Exec(ctx, tx); err != nil {
			return err
		}

		if len(adminRoles) == 0 {
			return errors.New("Administrator role not found - ensure RBAC service bootstrap has run")
		}

		adminRoleKey := adminRoles[0].Key

		// Get all users
		var users []User
		if err := s.NewRetrieve().Entries(&users).Exec(ctx, tx); err != nil {
			return err
		}

		// Assign Administrator role to all root users
		w := s.NewWriter(tx)
		rootUserCount := 0
		for _, u := range users {
			if u.RootUser {
				// Assign Administrator role
				if err := w.AssignRoles(ctx, u.Key, adminRoleKey); err != nil {
					return err
				}
				rootUserCount++
			}
		}

		// Mark migration as complete
		if err := tx.Set(ctx, rbacMigrationKey, rbacMigrationValue); err != nil {
			return err
		}

		return nil
	})
}
