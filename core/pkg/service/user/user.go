// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package user provides the user management for the Synnax cluster. user includes a
// service for managing users, a writer for creating and updating users, and a reader
// for reading users. Authentication and authorization of users is done separately in
// package auth.
package user

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

// A User is a representation of a user in the Synnax cluster.
type User struct {
	// Key is the unique identifier for the user.
	Key uuid.UUID `json:"key"`
	// Username is the unique username for the user. Username is also enforced to be
	// unique.
	Username string `json:"username"`
	// FirstName is the first name of the user.
	FirstName string `json:"first_name" msgpack:"first_name"`
	// LastName is the last name of the user.
	LastName string `json:"last_name" msgpack:"last_name"`
	// RootUser is a boolean that determines if the user is a root user. Root users are
	// the users that configure the Synnax server, and have full access to the server.
	// Deprecated: Use Roles field for role-based access control.
	RootUser bool `json:"root_user" msgpack:"root_user"`
	// Roles is the list of role UUIDs assigned to this user. Users receive permissions
	// from all policies associated with their assigned roles.
	Roles []uuid.UUID `json:"roles" msgpack:"roles"`
}

var _ gorp.Entry[uuid.UUID] = User{}

// GorpKey implements gorp.Entry.
func (u User) GorpKey() uuid.UUID { return u.Key }

// SetOptions implements gorp.Entry.
func (u User) SetOptions() []any { return nil }
