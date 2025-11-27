// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package example demonstrates jerky code generation.
package example

import (
	"time"

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// UserID is a custom type wrapper demonstrating how jerky handles type aliases.
type UserID uint32

//go:generate jerky
type User struct {
	// Key is the unique identifier for the user.
	Key uuid.UUID `json:"key"`
	// ID is a custom type wrapper (uint32 under the hood).
	ID UserID `json:"id"`
	// Name is the user's display name.
	Name string `json:"name"`
	// Email is the user's email address.
	Email string `json:"email"`
	// Age is the user's age in years.
	Age int32 `json:"age"`
	// Active indicates if the user account is active.
	Active bool `json:"active"`
	// Balance is the user's account balance.
	Balance float64 `json:"balance"`
	// CreatedAt is when the user was created.
	CreatedAt time.Time `json:"created_at"`
	// LastSeen is the last time the user was active (telem.TimeStamp is int64).
	LastSeen telem.TimeStamp `json:"last_seen"`
	// Tags are labels associated with the user.
	Tags []string `json:"tags"`
	// Role is a new field added in V2.
	Role string `json:"role"`
	// Verified indicates if the user's email is verified (new in V3).
	Verified bool `json:"verified"`
	// Score is the user's reputation score (new in V3).
	Score int64 `json:"score"`
	// Department is the user's department (new field for testing).
	Department string `json:"department"`
	// Address is the user's primary address (jerky embedded type).
	Address Address `json:"address"`
	// Addresses is a list of user addresses (slice of jerky embedded type).
	Addresses []Address `json:"addresses"`
}

var _ gorp.Entry[uuid.UUID] = User{}

func (u User) GorpKey() uuid.UUID { return u.Key }

func (u User) SetOptions() []any { return nil }
