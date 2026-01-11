// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

var _ gorp.Entry[uuid.UUID] = User{}

// GorpKey implements gorp.Entry.
func (u User) GorpKey() uuid.UUID { return u.Key }

// SetOptions implements gorp.Entry.
func (u User) SetOptions() []any { return nil }
