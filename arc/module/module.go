// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package module

import (
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
)

type Module struct {
	ir.IR
	compiler.Output
}

func (m Module) IsZero() bool {
	return len(m.Nodes) == 0 && len(m.Functions) == 0 && len(m.Edges) == 0 && len(m.WASM) == 0 && m.Symbols == nil
}
