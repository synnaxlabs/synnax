// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package directory

import (
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// OntologyID returns the ID of the directory with key in the ontology.
func OntologyID(key Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeArcDirectory, Key: key.String()}
}

// KeyFromOntologyID returns the directory key in id, or a validation error.
func KeyFromOntologyID(id ontology.ID) (Key, error) {
	if id.Type != ontology.ResourceTypeArcDirectory {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation, "must be an Arc directory, got %q", id.Type,
		)
	}
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation, "invalid Arc directory key %q: %v", id.Key, err,
		)
	}
	return key, nil
}
