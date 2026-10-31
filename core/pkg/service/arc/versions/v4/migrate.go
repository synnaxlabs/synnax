// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"context"

	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
)

// MigrateArc lifts a v3 Arc, rewriting its text from the grammar that wrote inputs in
// braces and triggers in parens. A stored Arc carries its text as a document and an
// exported one as raw; either way the result carries a fresh document and no raw.
func MigrateArc(ctx context.Context, old v3.Arc) (Arc, error) {
	a, err := autoMigrateArc(ctx, old)
	if err != nil {
		return Arc{}, err
	}
	raw := a.Text.Raw
	if len(a.Text.Doc.Inserts) > 0 {
		raw = a.Text.Materialize().Raw
	}
	if raw == "" {
		return a, nil
	}
	a.Text.Doc = text.Create(SwapBrackets(raw))
	a.Text.Raw = ""
	return a, nil
}

// Migration rewrites the text of every stored Arc for the bracket swap.
var Migration = migrate.NewEntry("v61_swap_brackets", MigrateArc)
