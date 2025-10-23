// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/x/gorp"
)

// Policy is the V0 (unversioned, legacy) policy structure from RC.
// This had direct subject attachment and NO version field.
// Preserved for migration purposes only.
type Policy struct {
	Key      uuid.UUID       `json:"key" msgpack:"key"`
	Subjects []ontology.ID   `json:"subjects" msgpack:"subjects"` // V0-specific
	Objects  []ontology.ID   `json:"objects" msgpack:"objects"`
	Actions  []access.Action `json:"actions" msgpack:"actions"`
}

var _ gorp.Entry[uuid.UUID] = Policy{}

func (p Policy) GorpKey() uuid.UUID { return p.Key }
func (p Policy) SetOptions() []any  { return nil }
func (p Policy) GetVersion() uint8  { return 0 }

// ToV1 converts a V0 policy to V1 format (without subjects, with version).
func (p Policy) ToV1() policy.Policy {
	return policy.Policy{
		Key:     p.Key,
		Name:    fmt.Sprintf("Migrated-%s", p.Key.String()[:8]),
		Effect:  policy.EffectAllow, // Default to allow
		Objects: p.Objects,
		Actions: p.Actions,
	}
}
