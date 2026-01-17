// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package access provides a framework for enforcing access control policies on
// resources in the Synnax ontology graph.
package access

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

var ErrDenied = errors.New("access denied")

type Action string

const (
	ActionCreate   Action = "create"
	ActionRetrieve Action = "retrieve"
	ActionUpdate   Action = "update"
	ActionDelete   Action = "delete"
)

var AllActions = []Action{ActionCreate, ActionRetrieve, ActionUpdate, ActionDelete}

// Request represents an access control request to be evaluated by an Enforcer. Not all
// fields are required for each access control operation. For instance, a TimeRange will
// only be required for data read/write operations, while Properties will only be
// required for update operations.
type Request struct {
	// Properties is the list of properties being modified (for update operations).
	Properties set.Set[string]
	// Subject is the entity (typically a user) making the request.
	Subject ontology.ID
	// Action is the type of action being performed (retrieve, create, etc.).
	Action Action
	// Objects are the resources being accessed.
	Objects []ontology.ID
	// TimeRange is the time range being accessed (for data read/write operations).
	TimeRange telem.TimeRange
}

// Enforcer evaluates access control requests.
type Enforcer interface {
	// Enforce checks if ALL requested objects are accessible. Returns ErrDenied if any
	// object in the request is not accessible, or nil if all objects are allowed. Use
	// this for transactional operations where partial access is not acceptable.
	Enforce(context.Context, Request) error

	// Filter returns the subset of requested objects that are accessible to the
	// subject. Unlike Enforce, Filter never returns ErrDenied - instead it returns an
	// empty set if no objects are accessible. Only returns an error for
	// system/configuration issues. Use this for discovery operations where partial
	// results are acceptable.
	Filter(context.Context, Request) (set.Set[ontology.ID], error)
}
