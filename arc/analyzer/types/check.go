// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func Check(
	cs *constraints.System,
	t1, t2 types.Type,
	source antlr.ParserRuleContext,
	reason string,
) error {
	if t1.Kind == types.KindTypeVariable || t2.Kind == types.KindTypeVariable {
		cs.AddEquality(t1, t2, source, reason)
		return nil
	}
	if t1.Kind == types.KindChan {
		if t2.Kind == types.KindChan {
			return Check(cs, t1.Unwrap(), t2.Unwrap(), source, reason+" (channel value types)")
		}
		return errors.Newf("type mismatch: expected %v, got %v", t1, t2)
	}
	if t1.Kind == types.KindSeries {
		if t2.Kind == types.KindSeries {
			return Check(cs, t1.Unwrap(), t2.Unwrap(), source, reason+" (series element types)")
		}
		return errors.Newf("type mismatch: expected %v, got %v", t1, t2)
	}
	if !types.Equal(t1, t2) {
		return errors.Newf("type mismatch: expected %v, got %v", t1, t2)
	}
	return nil
}
