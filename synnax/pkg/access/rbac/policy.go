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
// A policy sets an action that is allowed. All other accesses except for those
// specified by a policy are denied by default.
//
// In a policy, **Subjects do Actions on Objects**.
type Policy struct {
	// Key is a unique uuid to identify the policy.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Subjects it the list of subjects of the policy
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
	// Objects is the list of objects that the policy applies to
	Objects []ontology.ID `json:"objects" msgpack:"objects"`
	// Actions is the list of actions that the policy applies to
	Actions []access.Action `json:"actions" msgpack:"actions"`
}

var _ gorp.Entry[uuid.UUID] = Policy{}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() uuid.UUID { return p.Key }

// SetOptions implements the gorp.Entry interface.
func (p Policy) SetOptions() []interface{} { return nil }

// allowRequest returns true if the policies allow the given access.Request.
//
// For a request to be allowed:
//   - The request's subject must have object-action pairs for each object specified in
//     the request for the action specified in the request.
//   - An object-action pair is a pair with the specified action in the request and an
//     object that is either a type object with the correct type, or an object that
//     exactly matches one of the requested objects.
func allowRequest(req access.Request, policies []Policy) bool {
	requestedObjects := make(map[ontology.ID]struct{})
	for _, o := range req.Objects {
		requestedObjects[o] = struct{}{}
	}

	for _, policy := range policies {
		if !lo.Contains(policy.Subjects, req.Subject) {
			// Policy not directly pertaining to requested subject.
			var shouldContinue bool = true
			for _, s := range policy.Subjects {
				if s.IsType() && s.Type == req.Subject.Type {
					shouldContinue = false
					break
				}
			}
			if shouldContinue {
				continue
			}
		}
		if policy.Actions != nil && !lo.Contains(policy.Actions, req.Action) && !lo.Contains(policy.Actions, access.All) {
			// If the requested action is not described by the current policy,
			// skip the policy.
			// Unless the policy is an AllowAll, in which case do not skip.
			if !lo.Contains(policy.Objects, AllowAll) {
				continue
			}
		}

		for _, o := range policy.Objects {
			if o.Type == AllowAll.Type {
				// If the subject has an AllowAll policy, allow all requests.
				return true
			}
			if o.IsType() {
				// If an object applies to an entire type, then all requested objects
				// of that type may be satisfied.
				for requestedO := range requestedObjects {
					if requestedO.Type == o.Type {
						delete(requestedObjects, requestedO)
					}
				}
			} else {
				delete(requestedObjects, o)
			}
		}
	}

	return len(requestedObjects) == 0
}
