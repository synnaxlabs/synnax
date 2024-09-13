// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type Writer struct {
	svc *Service
	tx  gorp.Tx
	otg ontology.Writer
}

// Create makes a new user in the key- value store.
func (w Writer) Create(ctx context.Context, u *User) error {
	if u.Key == uuid.Nil {
		u.Key = uuid.New()
	}
	exists, err := w.svc.UsernameExists(ctx, u.Username)
	if err != nil {
		return err
	}
	if exists {
		return query.UniqueViolation
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

func (w Writer) ChangeUsername(ctx context.Context, key uuid.UUID, newUsername string) error {
	usernameExists, err := w.svc.UsernameExists(ctx, newUsername)
	if err != nil {
		return err
	}
	if usernameExists {
		return query.UniqueViolation
	}
	return gorp.NewUpdate[uuid.UUID, User]().WhereKeys(key).Change(func(u User) User {
		u.Username = newUsername
		return u
	}).Exec(ctx, w.tx)
}

// ChangeName updates the first and last name of the user with the given key. If either
// first or last is an empty string, the corresponding field will not be updated.
func (w Writer) ChangeName(ctx context.Context, key uuid.UUID, first string, last string) error {
	return gorp.NewUpdate[uuid.UUID, User]().WhereKeys(key).Change(func(u User) User {
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
	if err := gorp.NewDelete[uuid.UUID, User]().WhereKeys(keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteManyResources(ctx, OntologyIDsFromKeys(keys))
}
