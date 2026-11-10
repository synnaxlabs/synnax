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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
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

// Stamp sets ref.ICDHash to the TaskHash of the given messages in the referenced ICD
// and returns the ICD. When ref has no ICD, Stamp clears ref.ICDHash and returns the
// zero ICD. It returns a validation error scoped to the "icd" path when the ICD does
// not exist in snap. Task config stores call it on write, so it reads ICDs straight
// from snap and needs no Service, which opens after the task service.
func Stamp(
	ctx context.Context,
	snap gorp.Snapshot,
	ref *Reference,
	messages []MessageKey,
) (ICD, error) {
	if ref.ICD == nil {
		ref.ICDHash = ""
		return ICD{}, nil
	}
	var ic ICD
	if err := gorp.NewRetrieve[Key, ICD]().
		Where(gorp.MatchKeys[Key, ICD](*ref.ICD)).
		Entry(&ic).
		Exec(ctx, snap); err != nil {
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
	return ic, nil
}
