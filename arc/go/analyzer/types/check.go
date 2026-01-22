// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package types provides type inference and checking for Arc language analysis.
package types

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// Check verifies type compatibility between t1 and t2, adding constraints for type variables
// or recursively checking wrapped types for channels and series.
func Check(
	cs *constraints.System,
	t1, t2 types.Type,
	source antlr.ParserRuleContext,
	reason string,
) error {
	if t1.Kind == types.KindInvalid || t2.Kind == types.KindInvalid {
		return nil
	}

	if t1.Kind != types.KindVariable && t2.Kind != types.KindVariable {
		if !types.StructuralMatch(t1, t2) {
			return errors.Newf("type mismatch: expected %v, got %v", t1, t2)
		}
	}

	if t1.Kind == types.KindVariable || t2.Kind == types.KindVariable {
		cs.AddEquality(t1, t2, source, reason)
		return nil
	}

	if t1.Kind == types.KindSeries || t1.Kind == types.KindChan {
		return Check(cs, t1.Unwrap(), t2.Unwrap(), source, reason+" (element types)")
	}

	if !types.Equal(t1, t2) {
		return errors.Newf("type mismatch: expected %v, got %v", t1, t2)
	}
	return nil
}
