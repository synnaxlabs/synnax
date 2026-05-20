// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stl is the Arc standard library. It exposes a flat slice of
// symbols that callers attach to a program's ambient prelude via
// symbol.AttachToAmbient. The package owns no root-building or
// scope-assembly logic; that lives in the symbol package.
package stl

import (
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

// Symbols is the flattened set of symbols every STL package contributes
// to a program's ambient prelude.
var Symbols = func() []*symbol.Symbol {
	var out []*symbol.Symbol
	for _, pkg := range [][]*symbol.Symbol{
		channels.Symbols,
		constant.Symbols,
		control.Symbols,
		errors.Symbols,
		math.Symbols,
		op.Symbols,
		selector.Symbols,
		series.Symbols,
		stable.Symbols,
		stateful.Symbols,
		strings.Symbols,
		time.Symbols,
	} {
		out = append(out, pkg...)
	}
	return out
}()
