// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import (
	"github.com/synnaxlabs/slate/compiler/runtime"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/symbol"
)

// Context maintains compilation state across all code generation
type Context struct {
	Imports *runtime.ImportIndex
	Scope   *symbol.Scope
	Writer  *wasm.Writer
	Module  *wasm.Module
}
