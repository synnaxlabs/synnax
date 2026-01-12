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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// Type discriminates between constraint kinds for serialization.
type Type string

// EnforceParams provides the context needed for constraint evaluation.
type EnforceParams struct {
	// Request is the access request being evaluated. When enforcing, Request.Objects
	// contains a single element - the specific object being checked.
	Request access.Request
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

// TODO: think about serializing these as an interface vs. just a struct that has
// kind-specific fields that go unused for other kinds.
type Constraint2[Params any] struct {
	ConstraintType string
	Params         Params
}

// registry maps constraint types to factory functions for deserialization.
var registry = make(map[Type]func() Constraint)

// Register registers a constraint type for deserialization.
func Register(t Type, factory func() Constraint) { registry[t] = factory }

// Unmarshal deserializes a constraint using the provided decoder and type
// discriminator.
func Unmarshal(
	ctx context.Context,
	decoder binary.Decoder,
	data []byte,
) (Constraint, error) {
	var typed struct {
		Type Type `json:"type" msgpack:"type"`
	}
	if err := decoder.Decode(ctx, data, &typed); err != nil {
		return nil, err
	}
	factory, ok := registry[typed.Type]
	if !ok {
		return nil, errors.Newf("unknown constraint type: %s", typed.Type)
	}
	c := factory()
	if err := decoder.Decode(ctx, data, c); err != nil {
		return nil, err
	}
	return c, nil
}

// UnmarshalMany deserializes a slice of raw constraint data using the provided decoder.
func UnmarshalMany(
	ctx context.Context,
	decoder binary.Decoder,
	rawConstraints [][]byte,
) ([]Constraint, error) {
	constraints := make([]Constraint, len(rawConstraints))
	var err error
	for i, raw := range rawConstraints {
		if constraints[i], err = Unmarshal(ctx, decoder, raw); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal constraint %d", i)
		}
	}
	return constraints, nil
}
