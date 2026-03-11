// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text

import (
	"context"

	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/program"
)

// Compile generates WebAssembly bytecode from the provided IR.
//
// Returns a Module containing both the IR and the compiled WebAssembly output.
// Compiler options can be provided to customize the compilation process.
func Compile(
	ctx context.Context,
	ir ir.IR,
	opts ...compiler.Option,
) (program.Program, error) {
	o, err := compiler.Compile(ctx, ir, opts...)
	if err != nil {
		return program.Program{}, err
	}
	return program.Program{IR: ir, Output: o}, nil
}
