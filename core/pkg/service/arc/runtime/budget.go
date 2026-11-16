// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/x/errors"
	"github.com/tetratelabs/wazero/api"
)

// DefaultLoopBudget is the loop iterations and function calls that a program can run
// in one cycle by default.
const DefaultLoopBudget = 100_000_000

// ErrLoopBudget is returned when a program uses its whole loop budget in one cycle.
var ErrLoopBudget = errors.New("program ran too long")

// Budget sets and reads the loop budget of a guest compiled from Arc. Each loop
// iteration and function call in the guest takes one unit, and the guest traps when
// it takes the last one. The zero Budget belongs to a program with no guest and never
// runs out.
type Budget struct {
	global api.MutableGlobal
	limit  uint64
}

// NewBudget returns the Budget of guest, which gets limit units each cycle. It returns
// an error if guest does not export the loop budget global.
func NewBudget(guest api.Module, limit int64) (Budget, error) {
	global, ok := guest.ExportedGlobal(wasm.LoopBudgetExport).(api.MutableGlobal)
	if !ok {
		return Budget{}, errors.Newf(
			"guest does not export a mutable %q global",
			wasm.LoopBudgetExport,
		)
	}
	return Budget{global: global, limit: uint64(limit)}, nil
}

// Reset gives the guest its whole budget for a new cycle.
func (b Budget) Reset() {
	if b.global != nil {
		b.global.Set(b.limit)
	}
}

// Err returns an error wrapping ErrLoopBudget when the guest used its whole budget
// since the last Reset, and nil otherwise.
func (b Budget) Err() error {
	if b.global == nil || int64(b.global.Get()) > 0 {
		return nil
	}
	return errors.Wrapf(
		ErrLoopBudget,
		"it ran more than %d loop iterations and function calls in one cycle",
		b.limit,
	)
}
