// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

const (
	// TypeEmpty represents an error that hasn't been properly parsed or detected.
	// TypeEmpty errors typically represent a programming error.
	TypeEmpty = ""
	// TypeNil represents a nil error i.e. one that has not occurred.
	TypeNil = "nil"
	// TypeUnknown represents an error that was not registered with the ferrors package.
	TypeUnknown = "unknown"
	// TypeRoach represents an error type that was encoded using cockroachdb's errors package.
	// This is the default error type for errors that are not registered with the ferrors package,
	// and is used mostly for go-to-go communication.
	TypeRoach = "roach"
)
