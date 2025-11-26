// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/version"
)

type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// Policy is a simple access control policy in the RBAC model. A policy sets an action
// that is allowed. All other accesses except for those specified by a policy are denied
// by default.
//
// This is the V1 policy format (role-based, versioned) with NO Subjects field.
// Policies are attached to roles, and roles are assigned to users via ontology relationships.
type Policy struct {
	Name string `json:"name" msgpack:"name"`
	// Key is a unique uuid to identify the policy.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Effect sets whether the policy denies or allows the actions on the provided
	// objects.
	Effect Effect `json:"effect" msgpack:"effect"`
	// Objects is the list of objects that the policy applies to
	Objects []ontology.ID `json:"objects" msgpack:"objects"`
	// Actions is the list of actions that the policy applies to
	Actions []access.Action `json:"actions" msgpack:"actions"`
	// Version tracks the policy schema version (V1 = role-based)
	Version version.Counter `json:"version" msgpack:"version"`
	// Internal indicates whether the policy is built-in to the system.
	Internal bool `json:"internal" msgpack:"internal"`
}

var _ gorp.Entry[uuid.UUID] = Policy{}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() uuid.UUID { return p.Key }

// SetOptions implements the gorp.Entry interface.
func (p Policy) SetOptions() []any { return nil }
