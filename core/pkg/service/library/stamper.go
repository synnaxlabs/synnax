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
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Stamper records a task's use of a library. Task config stores call it on write, so
// it reads libraries straight from the database and needs no library Service, which
// opens after the task service.
type Stamper struct {
	// DB is the database libraries are stored in.
	DB *gorp.DB
	// Ontology relates the task to the library it uses.
	Ontology *ontology.Ontology
}

// Stamp sets ref.LibraryHash to the current hash of the referenced library, makes the
// library the only one the task uses, and returns the library. It returns a path-scoped
// validation error when the library does not exist. The task's ontology resource must
// exist.
func (s Stamper) Stamp(
	ctx context.Context,
	tx gorp.Tx,
	taskKey uuid.UUID,
	ref *Reference,
) (Library, error) {
	tx = gorp.OverrideTx(s.DB, tx)
	var l Library
	if err := gorp.NewRetrieve[Key, Library]().
		Where(gorp.MatchKeys[Key, Library](ref.Library)).
		Entry(&l).
		Exec(ctx, tx); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return l, validate.PathedError(errors.Wrapf(
				validate.ErrValidation, "library %s does not exist", ref.Library,
			), "library")
		}
		return l, err
	}
	hash, err := Hash(l)
	if err != nil {
		return l, err
	}
	ref.LibraryHash = hash
	return l, s.Ontology.NewWriter(tx).ReplaceOutgoingRelationshipsOfType(
		ctx,
		ontology.ID{Type: ontology.ResourceTypeTask, Key: taskKey.String()},
		ontology.RelationshipTypeUses,
		OntologyID(ref.Library),
	)
}
