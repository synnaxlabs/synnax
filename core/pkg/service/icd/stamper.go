// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Use is one message a task config uses, with the fields of it the task reads or
// writes.
type Use struct {
	// Message is the key of the message.
	Message MessageKey
	// Fields are the keys of the fields the task uses.
	Fields []FieldKey
}

// Stamper records a task's use of an ICD. Task config stores call it on write, so it
// reads ICDs straight from the database and needs no ICD Service, which opens after
// the task service.
type Stamper struct {
	// DB is the database ICDs are stored in.
	DB *gorp.DB
	// Ontology relates the task to the ICD it uses.
	Ontology *ontology.Ontology
}

// Stamp sets ref.ICDHash to the TaskHash of the given messages in the referenced ICD
// and returns the ICD. A task uses at most one ICD, so Stamp replaces any ICD the task
// used before. When ref has no ICD, Stamp clears ref.ICDHash, removes the ICD the task
// used, and returns the zero ICD. It returns a validation error scoped to the "icd"
// path when the ICD does not exist. The task's ontology resource must exist.
func (s Stamper) Stamp(
	ctx context.Context,
	tx gorp.Tx,
	taskKey uuid.UUID,
	ref *Reference,
	messages []MessageKey,
) (ICD, error) {
	tx = gorp.OverrideTx(s.DB, tx)
	w := s.Ontology.NewWriter(tx)
	taskID := task.OntologyID(taskKey)
	if ref.ICD == nil {
		ref.ICDHash = ""
		return ICD{}, library.Use(ctx, w, taskID, nil)
	}
	var ic ICD
	if err := gorp.NewRetrieve[Key, ICD]().
		Where(gorp.MatchKeys[Key, ICD](*ref.ICD)).
		Entry(&ic).
		Exec(ctx, tx); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return ic, validate.PathedError(errors.Wrapf(
				validate.ErrValidation, "ICD %s does not exist", *ref.ICD,
			), "icd")
		}
		return ic, err
	}
	hash, err := TaskHash(ic, messages)
	if err != nil {
		return ic, err
	}
	ref.ICDHash = hash
	return ic, library.Use(ctx, w, taskID, new(ic.OntologyID()))
}
