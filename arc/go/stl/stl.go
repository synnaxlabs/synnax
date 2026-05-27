// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stl is the Arc standard library. It exposes a flat slice of
// symbols that callers pass to symbol.NewRoot as ambient globals.
// The package owns no root-building or scope-assembly logic; that lives
// in the symbol package.
package stl

import (
	"slices"

	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/stl/constant"
	"github.com/synnaxlabs/arc/stl/control"
	"github.com/synnaxlabs/arc/stl/errors"
	"github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/op"
	"github.com/synnaxlabs/arc/stl/selector"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stable"
	"github.com/synnaxlabs/arc/stl/stateful"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/symbol"
)

// NewSymbols returns a fresh slice of every STL package's ambient prelude
// symbols. Each call allocates a new tree so concurrent analyses (e.g. the
// LSP serving multiple documents) and successive analyses on the same
// process never share mutable symbol state.
func NewSymbols() []*symbol.Symbol {
	return slices.Concat(
		channels.NewSymbols(),
		constant.NewSymbols(),
		control.NewSymbols(),
		errors.NewSymbols(),
		math.NewSymbols(),
		op.NewSymbols(),
		selector.NewSymbols(),
		series.NewSymbols(),
		stable.NewSymbols(),
		stateful.NewSymbols(),
		strings.NewSymbols(),
		time.NewSymbols(),
	)
}
