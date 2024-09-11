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

// Create makes a new user in the key-value store.
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

// Update updates the given user in the key-value store.
func (w Writer) Update(ctx context.Context, u User) error {
	return gorp.NewCreate[uuid.UUID, User]().Entry(&u).Exec(ctx, w.tx)
}

// Delete removes the users with the given keys from the key-value store.
func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	if err := gorp.NewDelete[uuid.UUID, User]().WhereKeys(keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, key := range keys {
		if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return nil
}
