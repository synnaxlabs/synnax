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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ParentKey converts opts.Parent to a project Key for importers whose resources are
// project children. Returns a validation error scoped to the "parent" field when the
// parent is not a project or its key is not a valid UUID.
func ParentKey(opts imex.ImportOptions) (Key, error) {
	if opts.Parent.Type != ontology.ResourceTypeProject {
		return uuid.Nil, validate.PathedError(
			errors.Wrapf(
				validate.ErrValidation,
				"parent must be a project, got %q",
				opts.Parent.Type,
			),
			"parent",
		)
	}
	key, err := uuid.Parse(opts.Parent.Key)
	if err != nil {
		return uuid.Nil, validate.PathedError(
			errors.Wrapf(
				validate.ErrValidation, "invalid project key %q", opts.Parent.Key,
			),
			"parent",
		)
	}
	return key, nil
}
