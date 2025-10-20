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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// A Writer is used to create, update, and delete users in the key-value store.
type Writer struct {
	// svc is the service that the writer is associated with. The service is needed to
	// check existing usernames in the key-value store.
	svc *Service
	// tx is the transaction that the writer will use to atomically interact with the
	// key-value.
	tx gorp.Tx
	// otg is the ontology writer that the writer will use to create relationships
	// between users and a user group.
	otg ontology.Writer
}

// Create makes a new user in the key-value store. If the username of u already exists,
// an error is thrown.
func (w Writer) Create(ctx context.Context, u *User) error {
	if u.Key == uuid.Nil {
		u.Key = uuid.New()
	}
	exists, err := w.svc.UsernameExists(ctx, u.Username)
	if err != nil {
		return err
	}
	if exists {
		return auth.RepeatedUsername
	}
	if err := gorp.NewCreate[uuid.UUID, User]().Entry(u).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := OntologyID(u.Key)
	if err = w.otg.DefineResource(ctx, otgID); err != nil {
		return err
	}
	return w.otg.DefineRelationship(ctx, w.svc.group.OntologyID(), ontology.ParentOf, otgID)
}

// ChangeUsername updates the username of the user with the given key. If a User with
// the username newUsername already exists, an error is thrown.
func (w Writer) ChangeUsername(ctx context.Context, key uuid.UUID, newUsername string) error {
	usernameExists, err := w.svc.UsernameExists(ctx, newUsername)
	if err != nil {
		return err
	}
	if usernameExists {
		return auth.RepeatedUsername
	}
	return gorp.NewUpdate[uuid.UUID, User]().WhereKeys(key).Change(func(_ gorp.Context, u User) User {
		u.Username = newUsername
		return u
	}).Exec(ctx, w.tx)
}

// ChangeName updates the first and last name of the user with the given key. If either
// first or last is an empty string, the corresponding field will not be updated.
func (w Writer) ChangeName(ctx context.Context, key uuid.UUID, first string, last string) error {
	return gorp.NewUpdate[uuid.UUID, User]().WhereKeys(key).Change(func(_ gorp.Context, u User) User {
		if first != "" {
			u.FirstName = first
		}
		if last != "" {
			u.LastName = last
		}
		return u
	}).Exec(ctx, w.tx)
}

// Delete removes the users with the given keys from the key-value store.
func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	if err := gorp.NewDelete[uuid.UUID, User]().WhereKeys(keys...).Guard(func(_ gorp.Context, u User) error {
		if u.RootUser {
			return errors.New("cannot delete root user")
		}
		return nil
	}).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteManyResources(ctx, OntologyIDsFromKeys(keys))
}

// MaybeSetRootUser will set the only user in the key-value store as the root user. This
// function is implemented to provide backwards compatibility with older versions of the
// key-value store from before v0.31.0.
func (w Writer) MaybeSetRootUser(
	ctx context.Context,
	username string,
) error {
	var users []User
	if err := w.svc.NewRetrieve().Entries(&users).Exec(ctx, w.tx); err != nil {
		return err
	}
	if len(users) != 1 {
		return nil
	}
	return gorp.NewUpdate[uuid.UUID, User]().WhereKeys(users[0].Key).Change(func(_ gorp.Context, u User) User {
		u.RootUser = true
		return u
	}).Exec(ctx, w.tx)
}

// AssignRoles assigns the given roles to a user. Duplicate role assignments are
// automatically deduplicated.
func (w Writer) AssignRoles(
	ctx context.Context,
	userKey uuid.UUID,
	roleKeys ...uuid.UUID,
) error {
	return gorp.NewUpdate[uuid.UUID, User]().WhereKeys(userKey).Change(func(_ gorp.Context, u User) User {
		// Create a set of existing roles for deduplication
		existingRoles := make(map[uuid.UUID]bool)
		for _, roleKey := range u.Roles {
			existingRoles[roleKey] = true
		}

		// Add new roles if they don't already exist
		for _, roleKey := range roleKeys {
			if !existingRoles[roleKey] {
				u.Roles = append(u.Roles, roleKey)
				existingRoles[roleKey] = true
			}
		}

		return u
	}).Exec(ctx, w.tx)
}

// UnassignRoles removes the given roles from a user.
func (w Writer) UnassignRoles(
	ctx context.Context,
	userKey uuid.UUID,
	roleKeys ...uuid.UUID,
) error {
	return gorp.NewUpdate[uuid.UUID, User]().WhereKeys(userKey).Change(func(_ gorp.Context, u User) User {
		// Create a set of roles to remove for efficient lookup
		toRemove := make(map[uuid.UUID]bool)
		for _, roleKey := range roleKeys {
			toRemove[roleKey] = true
		}

		// Filter out the roles to be removed
		newRoles := make([]uuid.UUID, 0, len(u.Roles))
		for _, roleKey := range u.Roles {
			if !toRemove[roleKey] {
				newRoles = append(newRoles, roleKey)
			}
		}

		u.Roles = newRoles
		return u
	}).Exec(ctx, w.tx)
}
