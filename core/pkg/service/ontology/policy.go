// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"context"

	"github.com/synnaxlabs/gorp"
)

// Policy guards one part of the ontology and returns nil outside it. Register it with
// [Ontology.RegisterPolicy].
type Policy interface {
	// ValidateParent is called before parent gains child as a parent_of child. Both
	// resources exist on tx.
	ValidateParent(ctx context.Context, tx gorp.Tx, parent, child ID) error
	// ValidateName is called before the resource with id takes name.
	ValidateName(ctx context.Context, tx gorp.Tx, id ID, name string) error
}
