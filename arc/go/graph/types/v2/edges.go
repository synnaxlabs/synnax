// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"github.com/samber/lo"
	ir "github.com/synnaxlabs/arc/ir/types/v2"
)

// IR projects the graph edges into their keyless ir.Edge form for compilation.
// The graph-layer key is an editing concern the compiler does not consume.
func (e Edges) IR() ir.Edges {
	return lo.Map(e, func(e Edge, _ int) ir.Edge { return e.Edge })
}
