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
	// ValidateParent is called before parent gains children as parent_of children, all
	// of them in one call. Every resource exists on tx.
	ValidateParent(ctx context.Context, tx gorp.Tx, parent ID, children ...ID) error
	// ValidateDetach is called before child loses parent_of parents, with the parents
	// it keeps.
	ValidateDetach(ctx context.Context, tx gorp.Tx, child ID, parents ...ID) error
	// ValidateName is called before the resource with id takes name.
	ValidateName(ctx context.Context, tx gorp.Tx, id ID, name string) error
}
