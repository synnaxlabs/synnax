// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

// Policy is a simple access control policy in the RBAC model.
type Policy struct {
	// Key is a unique uuid to identify the policy.
	Key uuid.UUID
	// Subjects it the list of subjects that the policy applies to
	Subjects []ontology.ID `json:"subject" msgpack:"subject"`
	// Objects is the list of objects that the policy applies to
	Objects []ontology.ID `json:"object" msgpack:"object"`
	// Actions is the list of actions that the policy applies to
	Actions []access.Action `json:"actions" msgpack:"actions"`
	// Effect is the effect of the policy. For example, a policy
	// could explicitly allow or deny access to the specified subject-object-action
	// combinations.
	Effect access.Effect `json:"effect" msgpack:"effect"`
}

var _ gorp.Entry[uuid.UUID] = Policy{}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() uuid.UUID { return p.Key }

// SetOptions implements the gorp.Entry interface.
func (p Policy) SetOptions() []interface{} { return nil }

// Matches returns true if the policy matches the given access.Request.
func (p Policy) Matches(req access.Request) bool {
	if !lo.Contains(p.Subjects, req.Subject) {
		return false
	}
	for _, filterObj := range req.Objects {
		if filterObj.IsType() {
			for _, obj := range p.Objects {
				if filterObj.Type != obj.Type {
					return false
				}
			}
		} else if !lo.Contains(p.Objects, filterObj) {
			return false
		}
	}
	return lo.Contains(p.Actions, req.Action)
}
