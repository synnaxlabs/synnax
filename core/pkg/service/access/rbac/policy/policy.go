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
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/gorp"
)

// Effect determines whether a policy allows or denies access.
type Effect string

const (
	// EffectAllow grants access when the policy matches.
	EffectAllow Effect = "allow"
	// EffectDeny denies access when the policy matches (takes precedence over allow).
	EffectDeny Effect = "deny"
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

// ConstraintType discriminates between constraint kinds for serialization.
type ConstraintType string

const (
	// ConstraintTypeField represents a field-based constraint.
	ConstraintTypeField ConstraintType = "field"
	// ConstraintTypeRelationship represents an ontology relationship constraint.
	ConstraintTypeRelationship ConstraintType = "relationship"
	// ConstraintTypeComputed represents a computed/derived value constraint.
	ConstraintTypeComputed ConstraintType = "computed"
)

// EnforceParams provides the context needed for constraint evaluation.
type EnforceParams struct {
	// Request is the access request being evaluated.
	Request access.Request
	// Object is the specific object being accessed.
	Object ontology.ID
	// Ontology provides access to the ontology graph for relationship lookups.
	Ontology *ontology.Ontology
	// Tx is the database transaction for queries.
	Tx gorp.Tx
}

// Constraint is the interface for policy constraints.
// Built-in types (Field, Relationship, Computed) implement this interface.
// Users can implement custom constraints for specialized logic.
type Constraint interface {
	// Enforce checks if the constraint is satisfied for the given request.
	// Returns nil if satisfied, or an error (typically access.ErrDenied) if not.
	Enforce(ctx context.Context, params EnforceParams) error
	// Type returns the constraint type for serialization.
	Type() ConstraintType
}

// FieldConstraint checks a stored value on a resource, subject, request, or system.
// Examples:
//   - resource.status == "active"
//   - subject.clearance in ["secret", "top_secret"]
//   - request.properties subset_of ["name", "description"]
type FieldConstraint struct {
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
func (c FieldConstraint) Type() ConstraintType { return ConstraintTypeField }

// RelationshipConstraint checks ontology graph relationships.
// Examples:
//   - resource created_by subject (creator-only access)
//   - resource labeled_by any of ["production", "safety-critical"]
type RelationshipConstraint struct {
	// Relationship is the ontology edge type to traverse.
	// Built-in types: "created_by", "labeled_by", "parent_of", "member_of"
	Relationship string `json:"relationship" msgpack:"relationship"`
	// Operator for comparison (eq_subject, in, contains_any, etc.)
	Operator Operator `json:"operator" msgpack:"operator"`
	// Value to compare against (can be omitted for eq_subject)
	Value any `json:"value,omitempty" msgpack:"value,omitempty"`
}

// Type implements Constraint.
func (c RelationshipConstraint) Type() ConstraintType { return ConstraintTypeRelationship }

// ComputedConstraint checks derived/calculated values.
// Examples:
//   - request.time_range duration <= 24h
//   - resource age > 30d
type ComputedConstraint struct {
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
func (c ComputedConstraint) Type() ConstraintType { return ConstraintTypeComputed }

// constraintRegistry maps constraint types to factory functions for deserialization.
var (
	constraintRegistry   = make(map[ConstraintType]func() Constraint)
	constraintRegistryMu sync.RWMutex
)

func init() {
	// Register built-in constraint types
	RegisterConstraintType(ConstraintTypeField, func() Constraint { return &FieldConstraint{} })
	RegisterConstraintType(ConstraintTypeRelationship, func() Constraint { return &RelationshipConstraint{} })
	RegisterConstraintType(ConstraintTypeComputed, func() Constraint { return &ComputedConstraint{} })
}

// RegisterConstraintType registers a custom constraint type for deserialization.
// This allows users to define custom constraint implementations that can be
// serialized and deserialized alongside built-in types.
func RegisterConstraintType(t ConstraintType, factory func() Constraint) {
	constraintRegistryMu.Lock()
	defer constraintRegistryMu.Unlock()
	constraintRegistry[t] = factory
}

// constraintWrapper is used for JSON serialization of constraints.
type constraintWrapper struct {
	Type ConstraintType `json:"type"`
	// Embed all possible fields - only relevant ones will be populated
	// FieldConstraint fields
	Target string   `json:"target,omitempty"`
	Field  []string `json:"field,omitempty"`
	// RelationshipConstraint fields
	Relationship string `json:"relationship,omitempty"`
	// ComputedConstraint fields
	Property string   `json:"property,omitempty"`
	Source   []string `json:"source,omitempty"`
	// Common fields
	Operator Operator `json:"operator,omitempty"`
	Value    any      `json:"value,omitempty"`
}

// MarshalJSON implements json.Marshaler for FieldConstraint.
func (c FieldConstraint) MarshalJSON() ([]byte, error) {
	return json.Marshal(constraintWrapper{
		Type:     ConstraintTypeField,
		Target:   c.Target,
		Field:    c.Field,
		Operator: c.Operator,
		Value:    c.Value,
	})
}

// MarshalJSON implements json.Marshaler for RelationshipConstraint.
func (c RelationshipConstraint) MarshalJSON() ([]byte, error) {
	return json.Marshal(constraintWrapper{
		Type:         ConstraintTypeRelationship,
		Relationship: c.Relationship,
		Operator:     c.Operator,
		Value:        c.Value,
	})
}

// MarshalJSON implements json.Marshaler for ComputedConstraint.
func (c ComputedConstraint) MarshalJSON() ([]byte, error) {
	return json.Marshal(constraintWrapper{
		Type:     ConstraintTypeComputed,
		Property: c.Property,
		Source:   c.Source,
		Operator: c.Operator,
		Value:    c.Value,
	})
}

// UnmarshalConstraint deserializes a JSON constraint using the type registry.
func UnmarshalConstraint(data []byte) (Constraint, error) {
	// First, unmarshal just the type field
	var wrapper constraintWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	// Look up the factory in the registry
	constraintRegistryMu.RLock()
	factory, ok := constraintRegistry[wrapper.Type]
	constraintRegistryMu.RUnlock()
	if !ok {
		return nil, &json.UnmarshalTypeError{
			Value: string(wrapper.Type),
			Type:  nil,
		}
	}

	// Create the constraint and populate it from the wrapper
	c := factory()
	switch constraint := c.(type) {
	case *FieldConstraint:
		constraint.Target = wrapper.Target
		constraint.Field = wrapper.Field
		constraint.Operator = wrapper.Operator
		constraint.Value = wrapper.Value
	case *RelationshipConstraint:
		constraint.Relationship = wrapper.Relationship
		constraint.Operator = wrapper.Operator
		constraint.Value = wrapper.Value
	case *ComputedConstraint:
		constraint.Property = wrapper.Property
		constraint.Source = wrapper.Source
		constraint.Operator = wrapper.Operator
		constraint.Value = wrapper.Value
	default:
		// For custom constraints, unmarshal directly
		if err := json.Unmarshal(data, c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// UnmarshalConstraints deserializes a JSON array of constraints.
func UnmarshalConstraints(data []byte) ([]Constraint, error) {
	var rawConstraints []json.RawMessage
	if err := json.Unmarshal(data, &rawConstraints); err != nil {
		return nil, err
	}
	constraints := make([]Constraint, len(rawConstraints))
	for i, raw := range rawConstraints {
		c, err := UnmarshalConstraint(raw)
		if err != nil {
			return nil, err
		}
		constraints[i] = c
	}
	return constraints, nil
}

// Policy is an access control policy in the RBAC model. A policy sets an action
// that is allowed or denied. All accesses not explicitly allowed by a policy are denied
// by default.
//
// Policies are attached to roles, and roles are assigned to users via ontology relationships.
type Policy struct {
	// Name is a human-readable name for the policy.
	Name string `json:"name" msgpack:"name"`
	// Key is a unique uuid to identify the policy.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Objects is the list of objects that the policy applies to.
	Objects []ontology.ID `json:"objects" msgpack:"objects"`
	// Actions is the list of actions that the policy applies to.
	Actions []access.Action `json:"actions" msgpack:"actions"`
	// Effect determines whether this policy allows or denies access. Defaults to allow.
	Effect Effect `json:"effect" msgpack:"effect"`
	// Constraints specifies additional conditions that must all be met for this policy to apply.
	// If empty, the policy applies unconditionally (based on objects/actions match).
	Constraints []Constraint `json:"constraints,omitempty" msgpack:"constraints,omitempty"`
	// Internal indicates whether the policy is built-in to the system.
	Internal bool `json:"internal" msgpack:"internal"`
}

var _ gorp.Entry[uuid.UUID] = Policy{}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() uuid.UUID { return p.Key }

// SetOptions implements the gorp.Entry interface.
func (p Policy) SetOptions() []any { return nil }

// policyJSON is used for custom JSON marshaling of Policy.
type policyJSON struct {
	Name        string            `json:"name"`
	Key         uuid.UUID         `json:"key"`
	Objects     []ontology.ID     `json:"objects"`
	Actions     []access.Action   `json:"actions"`
	Effect      Effect            `json:"effect"`
	Constraints []json.RawMessage `json:"constraints,omitempty"`
	Internal    bool              `json:"internal"`
}

// MarshalJSON implements json.Marshaler for Policy.
func (p Policy) MarshalJSON() ([]byte, error) {
	pj := policyJSON{
		Name:     p.Name,
		Key:      p.Key,
		Objects:  p.Objects,
		Actions:  p.Actions,
		Effect:   p.Effect,
		Internal: p.Internal,
	}
	if len(p.Constraints) > 0 {
		pj.Constraints = make([]json.RawMessage, len(p.Constraints))
		for i, c := range p.Constraints {
			data, err := json.Marshal(c)
			if err != nil {
				return nil, err
			}
			pj.Constraints[i] = data
		}
	}
	return json.Marshal(pj)
}

// UnmarshalJSON implements json.Unmarshaler for Policy.
func (p *Policy) UnmarshalJSON(data []byte) error {
	var pj policyJSON
	if err := json.Unmarshal(data, &pj); err != nil {
		return err
	}
	p.Name = pj.Name
	p.Key = pj.Key
	p.Objects = pj.Objects
	p.Actions = pj.Actions
	p.Effect = pj.Effect
	p.Internal = pj.Internal
	if len(pj.Constraints) > 0 {
		p.Constraints = make([]Constraint, len(pj.Constraints))
		for i, raw := range pj.Constraints {
			c, err := UnmarshalConstraint(raw)
			if err != nil {
				return err
			}
			p.Constraints[i] = c
		}
	}
	return nil
}
