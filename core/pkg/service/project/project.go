// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// KeyFromOntologyID converts id to the Key of the project it identifies. It returns an
// error wrapping validate.ErrValidation if id is not a project or its key is not a
// valid UUID. Callers that resolve a caller-supplied field scope the returned error to
// that field with validate.PathedError.
func KeyFromOntologyID(id ontology.ID) (Key, error) {
	if id.Type != ontology.ResourceTypeProject {
		return uuid.Nil, errors.Wrapf(
			validate.ErrValidation, "must be a project, got %q", id.Type,
		)
	}
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return uuid.Nil, errors.Wrapf(
			validate.ErrValidation, "invalid project key %q", id.Key,
		)
	}
	return key, nil
}
