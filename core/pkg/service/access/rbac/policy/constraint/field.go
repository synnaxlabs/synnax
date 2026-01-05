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
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/telem"
)

// Operator defines how a constraint compares values.
type Operator string

const (
	// OpEqual checks if the value equals the expected value.
	OpEqual Operator = "eq"
	// OpNotEqual checks if the value does not equal the expected value.
	OpNotEqual Operator = "not_eq"
	// OpEqualSubject checks if the value equals the requesting subject.
	OpEqualSubject Operator = "eq_subject"
	// OpIn checks if the value is in the expected list.
	OpIn Operator = "in"
	// OpNotIn checks if the value is not in the expected list.
	OpNotIn Operator = "not_in"
	// OpContains checks if the list contains the expected value.
	OpContains Operator = "contains"
	// OpContainsAny checks if the list contains any of the expected values.
	OpContainsAny Operator = "contains_any"
	// OpWithin checks if the time range is within the expected range.
	OpWithin Operator = "within"
	// OpSubsetOf checks if the list is a subset of the expected list.
	OpSubsetOf Operator = "subset_of"
	// OpLessThan checks if the value is less than the expected value.
	OpLessThan Operator = "lt"
	// OpLessThanOrEqual checks if the value is less than or equal to the expected value.
	OpLessThanOrEqual Operator = "lte"
	// OpGreaterThan checks if the value is greater than the expected value.
	OpGreaterThan Operator = "gt"
	// OpGreaterThanOrEqual checks if the value is greater than or equal to the expected value.
	OpGreaterThanOrEqual Operator = "gte"
)

// Field checks a stored value on a resource, subject, request, or system.
// Examples:
//   - resource.status == "active"
//   - subject.clearance in ["secret", "top_secret"]
//   - request.properties subset_of ["name", "description"]
type Field struct {
	// Target is the namespace: "resource", "subject", "request", "system"
	Target string `json:"target" msgpack:"target"`
	// Field is the path to the field within the target.
	// e.g., ["status"] or ["status", "variant"] for nested fields
	Field []string `json:"field" msgpack:"field"`
	// Operator for comparison
	Operator Operator `json:"operator" msgpack:"operator"`
	// Value to compare against
	Value any `json:"value" msgpack:"value"`
}

// Type implements Constraint.
func (c Field) Type() Type { return TypeField }

// MarshalJSON implements json.Marshaler.
func (c Field) MarshalJSON() ([]byte, error) {
	type fc Field
	return json.Marshal(struct {
		Type Type `json:"type"`
		fc
	}{Type: TypeField, fc: fc(c)})
}

// Enforce checks if the constraint is satisfied.
func (c Field) Enforce(ctx context.Context, params EnforceParams) bool {
	actual, ok := resolveFieldValue(ctx, params, c.Target, c.Field)
	if !ok {
		return false
	}
	return applyOperator(c.Operator, actual, c.Value, params.Request.Subject)
}

// resolveFieldValue resolves a field path to its actual value.
func resolveFieldValue(ctx context.Context, params EnforceParams, target string, field []string) (any, bool) {
	switch target {
	case "request":
		return resolveRequestField(params.Request, field)
	case "resource":
		return resolveResourceField(ctx, params, field)
	case "subject":
		return resolveSubjectField(ctx, params, field)
	case "system":
		return resolveSystemField(field)
	default:
		return nil, false
	}
}

func resolveRequestField(req access.Request, field []string) (any, bool) {
	if len(field) == 0 || req.Context == nil {
		return nil, false
	}
	value, ok := req.Context[field[0]]
	return value, ok
}

func resolveResourceField(ctx context.Context, params EnforceParams, field []string) (any, bool) {
	if len(field) == 0 {
		return nil, false
	}
	var resources []ontology.Resource
	if err := params.Ontology.NewRetrieve().
		WhereIDs(params.Object).
		Entries(&resources).
		Exec(ctx, params.Tx); err != nil || len(resources) == 0 {
		return nil, false
	}
	if resources[0].Data == nil {
		return nil, false
	}
	return getNestedField(resources[0].Data, field)
}

func resolveSubjectField(ctx context.Context, params EnforceParams, field []string) (any, bool) {
	if len(field) == 0 {
		return params.Request.Subject, true
	}
	var resources []ontology.Resource
	if err := params.Ontology.NewRetrieve().
		WhereIDs(params.Request.Subject).
		Entries(&resources).
		Exec(ctx, params.Tx); err != nil || len(resources) == 0 {
		return nil, false
	}
	if resources[0].Data == nil {
		return nil, false
	}
	return getNestedField(resources[0].Data, field)
}

func resolveSystemField(field []string) (any, bool) {
	if len(field) == 0 {
		return nil, false
	}
	switch field[0] {
	case "current_time":
		return telem.Now(), true
	default:
		return nil, false
	}
}

func getNestedField(data any, path []string) (any, bool) {
	if len(path) == 0 {
		return data, true
	}
	dataMap, ok := data.(map[string]any)
	if !ok {
		return nil, false
	}
	value, ok := dataMap[path[0]]
	if !ok {
		return nil, false
	}
	if len(path) == 1 {
		return value, true
	}
	return getNestedField(value, path[1:])
}
