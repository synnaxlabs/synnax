// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

// compileIdentifier compiles variable references
func (e *Compiler) compileIdentifier(name string) (types.Type, error) {
	// Look up in local variables
	if idx, ok := e.ctx.GetLocal(name); ok {
		e.encoder.WriteLocalGet(idx)

		// Get type from symbol table
		// TODO: Get actual type from symbol table
		return types.F64{}, nil // Placeholder
	}

	// TODO: Check for channels, functions, etc.

	return nil, errors.Newf("undefined identifier: %s", name)
}
