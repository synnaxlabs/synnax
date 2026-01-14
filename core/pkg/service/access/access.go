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

type Source string

const (
	SourceConsole Source = "Console"
	SourcePluto   Source = "Pluto"
	SourceDriver  Source = "Driver"
)

// Request represents an access control request to be evaluated by an Enforcer. Not all
// fields are required for each access control operation. For instance, a TimeRange will
// only be required for data read/write operations, while Properties will only be
// required for update operations.
type Request struct {
	// Subject is the entity (typically a user) making the request.
	Subject ontology.ID
	// Objects are the resources being accessed.
	Objects []ontology.ID
	// Action is the type of action being performed (retrieve, create, etc.).
	Action Action
	// TimeRange is the time range being accessed (for data read/write operations).
	TimeRange telem.TimeRange
	// Properties is the list of properties being modified (for update operations).
	Properties []string
	// Source identifies the type of client making the request.
	Source Source
}

// Enforcer evaluates access control requests.
type Enforcer interface {
	// Enforce checks if the request is allowed. Returns ErrDenied if an object in the
	// request is not accessible.
	Enforce(context.Context, Request) error
}
