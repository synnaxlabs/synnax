// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package constraint defines the constraint types used in access control policies.
package constraint

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/gorp"
)

// Type discriminates between constraint kinds for serialization.
type Type string

const (
	// TypeField represents a field-based constraint.
	TypeField Type = "field"
	// TypeRelationship represents an ontology relationship constraint.
	TypeRelationship Type = "relationship"
	// TypeComputed represents a computed/derived value constraint.
	TypeComputed Type = "computed"
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
	// Returns true if satisfied, or false if not.
	Enforce(context.Context, EnforceParams) bool
	// Type returns the constraint type for serialization.
	Type() Type
}

// registry maps constraint types to factory functions for deserialization.
var (
	registry   = make(map[Type]func() Constraint)
	registryMu sync.RWMutex
)

// Register registers a custom constraint type for deserialization.
// This allows users to define custom constraint implementations that can be
// serialized and deserialized alongside built-in types.
func Register(t Type, factory func() Constraint) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[t] = factory
}

// Unmarshal deserializes a JSON constraint using the type discriminator.
func Unmarshal(data []byte) (Constraint, error) {
	var typed struct {
		Type Type `json:"type"`
	}
	if err := json.Unmarshal(data, &typed); err != nil {
		return nil, err
	}

	switch typed.Type {
	case TypeField:
		var c Field
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	case TypeRelationship:
		var c Relationship
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	case TypeComputed:
		var c Computed
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return c, nil
	default:
		// For custom constraints, use the registry
		registryMu.RLock()
		factory, ok := registry[typed.Type]
		registryMu.RUnlock()
		if !ok {
			return nil, &json.UnmarshalTypeError{Value: string(typed.Type), Type: nil}
		}
		c := factory()
		if err := json.Unmarshal(data, c); err != nil {
			return nil, err
		}
		return c, nil
	}
}

// UnmarshalMany deserializes a JSON array of constraints.
func UnmarshalMany(data []byte) ([]Constraint, error) {
	var rawConstraints []json.RawMessage
	if err := json.Unmarshal(data, &rawConstraints); err != nil {
		return nil, err
	}
	constraints := make([]Constraint, len(rawConstraints))
	for i, raw := range rawConstraints {
		c, err := Unmarshal(raw)
		if err != nil {
			return nil, err
		}
		constraints[i] = c
	}
	return constraints, nil
}
