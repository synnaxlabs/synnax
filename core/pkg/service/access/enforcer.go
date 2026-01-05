// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
)

// ContextKey constants for well-known request context values. These can be used as keys
// in Request.Context for common constraint scenarios.
const (
	// CtxTimeRange is the time range being accessed (for data read/write operations).
	// Value type: telem.TimeRange
	CtxTimeRange = "time_range"
	// CtxProperties is the list of properties being modified (for update operations).
	// Value type: []string
	CtxProperties = "properties"
	// CtxSource identifies the client making the request (e.g., "console", "pluto",
	// "driver"). Value type: string
	CtxSource = "source"
)

// Request represents an access control request to be evaluated by an Enforcer.
type Request struct {
	// Subject is the entity (typically a user) making the request.
	Subject ontology.ID
	// Objects are the resources being accessed.
	Objects []ontology.ID
	// Action is the type of access being requested (retrieve, create, update, delete).
	Action Action
	// Context holds arbitrary request-specific data that constraints can evaluate.
	// Use the Ctx* constants for well-known keys, or define custom keys as needed.
	Context map[string]any
}

// Enforcer evaluates access control requests.
type Enforcer interface {
	// Enforce checks if the request is allowed. Returns ErrDenied if ANY object in the
	// request is not accessible. Use this for operations that should fail completely if
	// access is denied (e.g., update, delete).
	Enforce(context.Context, Request) error

	// Filter returns only the objects from the request that the subject has access to.
	// Unlike Enforce, this does not fail on denied objects - it simply excludes them.
	// Use this for search/list operations where partial results are acceptable.
	Filter(context.Context, Request) ([]ontology.ID, error)
}
