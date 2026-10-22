// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library

import (
	"context"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Use records that user uses the library document doc, or no document when doc is nil.
// A resource uses at most one library document, so Use replaces the one user used
// before.
func Use(
	ctx context.Context,
	w ontology.Writer,
	user ontology.ID,
	doc *ontology.ID,
) error {
	if doc == nil {
		return w.DeleteOutgoingRelationshipsOfType(
			ctx,
			user,
			ontology.RelationshipTypeUses,
		)
	}
	return w.ReplaceOutgoingRelationshipsOfType(
		ctx,
		user,
		ontology.RelationshipTypeUses,
		*doc,
	)
}

// Users returns the resources that use the library document doc. A document with no
// ontology resource has no users.
func Users(
	ctx context.Context,
	tx gorp.Tx,
	otg *ontology.Ontology,
	doc ontology.ID,
) ([]ontology.Resource, error) {
	var users []ontology.Resource
	if err := otg.NewRetrieve().
		WhereIDs(doc).
		TraverseTo(ontology.UsersTraverser).
		Entries(&users).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	return users, nil
}

// CheckUnused returns an error wrapping validate.ErrValidation, naming the users of
// the library document doc, when any resource uses it. Domains call it before deleting
// a document.
func CheckUnused(
	ctx context.Context,
	tx gorp.Tx,
	otg *ontology.Ontology,
	doc ontology.ID,
) error {
	users, err := Users(ctx, tx, otg, doc)
	if err != nil || len(users) == 0 {
		return err
	}
	var res ontology.Resource
	if err = otg.NewRetrieve().WhereIDs(doc).Entry(&res).Exec(ctx, tx); err != nil {
		return err
	}
	names := lo.Map(users, func(r ontology.Resource, _ int) string { return r.Name })
	return errors.Wrapf(
		validate.ErrValidation,
		"%s is used by %s",
		res.Name,
		strings.Join(names, ", "),
	)
}
