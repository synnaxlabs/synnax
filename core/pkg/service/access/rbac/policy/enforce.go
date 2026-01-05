// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"context"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Enforce implements Constraint for FieldConstraint.
// It checks stored values on resources, subjects, requests, or system state.
func (c FieldConstraint) Enforce(ctx context.Context, params EnforceParams) error {
	actual, ok := resolveFieldValue(ctx, params, c.Target, c.Field)
	if !ok {
		return access.ErrDenied
	}
	if !applyOperator(c.Operator, actual, c.Value, params.Request.Subject) {
		return access.ErrDenied
	}
	return nil
}

// Enforce implements Constraint for RelationshipConstraint.
// It checks ontology graph relationships between resources.
func (c RelationshipConstraint) Enforce(ctx context.Context, params EnforceParams) error {
	// Get related entities via the ontology relationship
	related, ok := resolveRelationship(ctx, params, c.Relationship)
	if !ok {
		return access.ErrDenied
	}
	if !applyOperator(c.Operator, related, c.Value, params.Request.Subject) {
		return access.ErrDenied
	}
	return nil
}

// Enforce implements Constraint for ComputedConstraint.
// It checks derived/calculated values.
func (c ComputedConstraint) Enforce(ctx context.Context, params EnforceParams) error {
	computed, ok := computeValue(ctx, params, c.Property, c.Source)
	if !ok {
		return access.ErrDenied
	}
	if !applyOperator(c.Operator, computed, c.Value, params.Request.Subject) {
		return access.ErrDenied
	}
	return nil
}

// resolveFieldValue resolves a field path to its actual value.
func resolveFieldValue(
	ctx context.Context,
	params EnforceParams,
	target string,
	field []string,
) (any, bool) {
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

// resolveRequestField resolves a field from the request.
func resolveRequestField(req access.Request, field []string) (any, bool) {
	if len(field) == 0 {
		return nil, false
	}
	switch field[0] {
	case "time_range":
		if req.TimeRange.IsZero() {
			return nil, false
		}
		return req.TimeRange, true
	case "properties":
		if len(req.Properties) == 0 {
			return nil, false
		}
		return req.Properties, true
	default:
		return nil, false
	}
}

// resolveResourceField resolves a field from the resource being accessed.
func resolveResourceField(
	ctx context.Context,
	params EnforceParams,
	field []string,
) (any, bool) {
	if len(field) == 0 {
		return nil, false
	}
	// Fetch the resource from the ontology
	var resources []ontology.Resource
	if err := params.Ontology.NewRetrieve().
		WhereIDs(params.Object).
		Entries(&resources).
		Exec(ctx, params.Tx); err != nil || len(resources) == 0 {
		return nil, false
	}
	resource := resources[0]

	// Try to get the field from the resource's data
	if resource.Data == nil {
		return nil, false
	}
	return getNestedField(resource.Data, field)
}

// resolveSubjectField resolves a field from the requesting subject.
func resolveSubjectField(
	ctx context.Context,
	params EnforceParams,
	field []string,
) (any, bool) {
	if len(field) == 0 {
		// Return the subject ID itself
		return params.Request.Subject, true
	}
	// Fetch the subject from the ontology
	var resources []ontology.Resource
	if err := params.Ontology.NewRetrieve().
		WhereIDs(params.Request.Subject).
		Entries(&resources).
		Exec(ctx, params.Tx); err != nil || len(resources) == 0 {
		return nil, false
	}
	resource := resources[0]

	// Try to get the field from the subject's data
	if resource.Data == nil {
		return nil, false
	}
	return getNestedField(resource.Data, field)
}

// resolveSystemField resolves a system state field.
func resolveSystemField(field []string) (any, bool) {
	if len(field) == 0 {
		return nil, false
	}
	switch field[0] {
	case "mode":
		// TODO: Implement system mode retrieval from config/state
		return nil, false
	case "current_time":
		return telem.Now(), true
	default:
		return nil, false
	}
}

// getNestedField retrieves a nested field from a data structure.
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

// resolveRelationship resolves an ontology relationship to get related entities.
func resolveRelationship(
	ctx context.Context,
	params EnforceParams,
	relationship string,
) (any, bool) {
	relType := ontology.RelationshipType(relationship)

	// Query relationships directly from gorp where the object is the From
	var relationships []ontology.Relationship
	if err := gorp.NewRetrieve[[]byte, ontology.Relationship]().
		Where(func(_ gorp.Context, rel *ontology.Relationship) (bool, error) {
			return rel.From == params.Object && rel.Type == relType, nil
		}).
		Entries(&relationships).
		Exec(ctx, params.Tx); err != nil {
		return nil, false
	}

	// Collect related IDs (the "To" side of the relationship)
	var relatedIDs []ontology.ID
	for _, rel := range relationships {
		relatedIDs = append(relatedIDs, rel.To)
	}

	// If looking for a single relationship (like created_by), return the first match
	if len(relatedIDs) == 1 {
		return relatedIDs[0], true
	}
	if len(relatedIDs) > 1 {
		return relatedIDs, true
	}
	return nil, false
}

// computeValue computes a derived value from a source.
func computeValue(
	ctx context.Context,
	params EnforceParams,
	property string,
	source []string,
) (any, bool) {
	// First resolve the source value
	if len(source) < 1 {
		return nil, false
	}
	target := source[0]
	field := source[1:]
	sourceValue, ok := resolveFieldValue(ctx, params, target, field)
	if !ok {
		return nil, false
	}

	// Apply the computation
	switch property {
	case "duration":
		// Compute duration from a time range
		tr, ok := sourceValue.(telem.TimeRange)
		if !ok {
			return nil, false
		}
		return tr.Span(), true
	case "age":
		// Compute age from a timestamp (time elapsed since the timestamp)
		ts, ok := sourceValue.(telem.TimeStamp)
		if !ok {
			return nil, false
		}
		return ts.Span(telem.Now()), true
	case "count":
		// Count elements in a list
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

// applyOperator applies the comparison operator to the actual and expected values.
func applyOperator(
	op Operator,
	actual any,
	expected any,
	subject ontology.ID,
) bool {
	switch op {
	case OpEqual:
		return actual == expected

	case OpNotEqual:
		return actual != expected

	case OpEqualSubject:
		if id, ok := actual.(ontology.ID); ok {
			return id == subject
		}
		return false

	case OpIn:
		expectedList, ok := toStringSlice(expected)
		if !ok {
			return false
		}
		actualStr, ok := actual.(string)
		if !ok {
			return false
		}
		return lo.Contains(expectedList, actualStr)

	case OpNotIn:
		expectedList, ok := toStringSlice(expected)
		if !ok {
			return false
		}
		actualStr, ok := actual.(string)
		if !ok {
			return false
		}
		return !lo.Contains(expectedList, actualStr)

	case OpContains:
		actualList, ok := toStringSlice(actual)
		if !ok {
			return false
		}
		expectedStr, ok := expected.(string)
		if !ok {
			return false
		}
		return lo.Contains(actualList, expectedStr)

	case OpContainsAny:
		actualList, ok := toStringSlice(actual)
		if !ok {
			return false
		}
		expectedList, ok := toStringSlice(expected)
		if !ok {
			return false
		}
		for _, exp := range expectedList {
			if lo.Contains(actualList, exp) {
				return true
			}
		}
		return false

	case OpWithin:
		actualTR, ok := actual.(telem.TimeRange)
		if !ok {
			return false
		}
		expectedTR, ok := expected.(telem.TimeRange)
		if !ok {
			return false
		}
		return expectedTR.ContainsRange(actualTR)

	case OpSubsetOf:
		actualList, ok := toStringSlice(actual)
		if !ok {
			return false
		}
		expectedList, ok := toStringSlice(expected)
		if !ok {
			return false
		}
		for _, item := range actualList {
			if !lo.Contains(expectedList, item) {
				return false
			}
		}
		return true

	case OpLessThan:
		return compareNumeric(actual, expected) < 0

	case OpLessThanOrEqual:
		return compareNumeric(actual, expected) <= 0

	case OpGreaterThan:
		return compareNumeric(actual, expected) > 0

	case OpGreaterThanOrEqual:
		return compareNumeric(actual, expected) >= 0

	default:
		return false
	}
}

// toStringSlice attempts to convert a value to a string slice.
func toStringSlice(v any) ([]string, bool) {
	switch val := v.(type) {
	case []string:
		return val, true
	case []any:
		result := make([]string, len(val))
		for i, item := range val {
			str, ok := item.(string)
			if !ok {
				return nil, false
			}
			result[i] = str
		}
		return result, true
	default:
		return nil, false
	}
}

// compareNumeric compares two numeric values.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareNumeric(a, b any) int {
	// Handle time.Duration
	aDur, aIsDur := a.(time.Duration)
	bDur, bIsDur := b.(time.Duration)
	if aIsDur && bIsDur {
		if aDur < bDur {
			return -1
		} else if aDur > bDur {
			return 1
		}
		return 0
	}

	// Handle telem.TimeSpan
	aSpan, aIsSpan := a.(telem.TimeSpan)
	bSpan, bIsSpan := b.(telem.TimeSpan)
	if aIsSpan && bIsSpan {
		if aSpan < bSpan {
			return -1
		} else if aSpan > bSpan {
			return 1
		}
		return 0
	}

	// Handle integers
	aInt, aIsInt := toInt64(a)
	bInt, bIsInt := toInt64(b)
	if aIsInt && bIsInt {
		if aInt < bInt {
			return -1
		} else if aInt > bInt {
			return 1
		}
		return 0
	}

	// Handle floats
	aFloat, aIsFloat := toFloat64(a)
	bFloat, bIsFloat := toFloat64(b)
	if aIsFloat && bIsFloat {
		if aFloat < bFloat {
			return -1
		} else if aFloat > bFloat {
			return 1
		}
		return 0
	}

	// Can't compare, return 0 (equal) which will cause comparison to fail
	return 0
}

func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case uint:
		return int64(val), true
	case uint32:
		return int64(val), true
	case uint64:
		return int64(val), true
	default:
		return 0, false
	}
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float32:
		return float64(val), true
	case float64:
		return val, true
	default:
		return 0, false
	}
}
