// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint

import (
	"context"
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/telem"
)

// Computed checks derived/calculated values.
// Examples:
//   - request.time_range duration <= 24h
//   - resource age > 30d
type Computed struct {
	// Property is the computed value to evaluate.
	// Built-in: "duration", "age", "count"
	Property string `json:"property" msgpack:"property"`
	// Source is what to compute from (e.g., ["request", "time_range"])
	Source []string `json:"source" msgpack:"source"`
	// Operator for comparison
	Operator Operator `json:"operator" msgpack:"operator"`
	// Value to compare against
	Value any `json:"value" msgpack:"value"`
}

// Type implements Constraint.
func (c Computed) Type() Type { return TypeComputed }

// MarshalJSON implements json.Marshaler.
func (c Computed) MarshalJSON() ([]byte, error) {
	type cc Computed
	return json.Marshal(struct {
		Type Type `json:"type"`
		cc
	}{Type: TypeComputed, cc: cc(c)})
}

// Enforce checks if the constraint is satisfied.
func (c Computed) Enforce(ctx context.Context, params EnforceParams) bool {
	computed, ok := computeValue(ctx, params, c.Property, c.Source)
	if !ok {
		return false
	}
	return applyOperator(c.Operator, computed, c.Value, params.Request.Subject)
}

func computeValue(ctx context.Context, params EnforceParams, property string, source []string) (any, bool) {
	if len(source) < 1 {
		return nil, false
	}
	target := source[0]
	field := source[1:]
	sourceValue, ok := resolveFieldValue(ctx, params, target, field)
	if !ok {
		return nil, false
	}

	switch property {
	case "duration":
		tr, ok := sourceValue.(telem.TimeRange)
		if !ok {
			return nil, false
		}
		return tr.Span(), true
	case "age":
		ts, ok := sourceValue.(telem.TimeStamp)
		if !ok {
			return nil, false
		}
		return ts.Span(telem.Now()), true
	case "count":
		switch v := sourceValue.(type) {
		case []string:
			return len(v), true
		case []any:
			return len(v), true
		case []ontology.ID:
			return len(v), true
		default:
			return nil, false
		}
	default:
		return nil, false
	}
}
