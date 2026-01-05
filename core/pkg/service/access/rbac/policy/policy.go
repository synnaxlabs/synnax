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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/x/binary"
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

// Policy is an access control policy in the RBAC model. A policy sets an action that is
// allowed or denied. All accesses not explicitly allowed by a policy are denied by
// default.
//
// Policies are attached to roles, and roles are assigned to users via ontology
// relationships.
type Policy struct {
	// Name is a human-readable name for the policy.
	Name string `json:"name" msgpack:"name"`
	// Key is a unique uuid to identify the policy.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Objects is the list of objects that the policy applies to.
	Objects []ontology.ID `json:"objects" msgpack:"objects"`
	// Actions is the list of actions that the policy applies to.
	Actions []access.Action `json:"actions" msgpack:"actions"`
	// Effect determines whether this policy allows or denies access.
	Effect Effect `json:"effect" msgpack:"effect"`
	// Constraints specifies additional conditions that must all be met for this policy
	// to apply. If empty, the policy applies unconditionally (based on objects/actions
	// match).
	Constraints []constraint.Constraint `json:"constraints,omitempty" msgpack:"constraints,omitempty"`
	// Internal indicates whether the policy is built-in to the system.
	//
	// TODO: remove this and replace with ontology relationship created_by.
	Internal bool `json:"internal" msgpack:"internal"`
}

var _ gorp.Entry[uuid.UUID] = Policy{}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() uuid.UUID { return p.Key }

// SetOptions implements the gorp.Entry interface.
func (p Policy) SetOptions() []any { return nil }

// UnmarshalJSON implements json.Unmarshaler for Policy.
// Custom unmarshaling is only needed because Constraints is an interface slice.
func (p *Policy) UnmarshalJSON(data []byte) error {
	type Alias Policy
	aux := &struct {
		*Alias
		Constraints []json.RawMessage `json:"constraints,omitempty"`
	}{Alias: (*Alias)(p)}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	p.Constraints = nil
	dec := &binary.JSONCodec{}
	for _, raw := range aux.Constraints {
		c, err := constraint.Unmarshal(context.Background(), dec, raw)
		if err != nil {
			return err
		}
		p.Constraints = append(p.Constraints, c)
	}
	return nil
}
