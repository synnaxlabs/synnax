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
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
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
	// Effect determines whether this policy allows or denies access.
	Effect Effect `json:"effect" msgpack:"effect"`
	// Constraint specifies the condition that must be met for this policy to apply. The
	// constraint contains the Objects and Actions that the policy applies to, as well
	// as any additional conditions (field checks, relationship checks, etc.). Use
	// logical constraint kinds (And, Or, Not) to combine multiple conditions.
	Constraint constraint.Constraint `json:"constraint" msgpack:"constraint"`
}

var _ gorp.Entry[uuid.UUID] = Policy{}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() uuid.UUID { return p.Key }

// SetOptions implements the gorp.Entry interface.
func (p Policy) SetOptions() []any { return nil }
