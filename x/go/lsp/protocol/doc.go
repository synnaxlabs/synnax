// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package protocol implements Language Server Protocol specification in Go.
//
// This package contains the structs that map directly to the wire format
// of the Language Server Protocol.
//
// It is a literal transcription, with unmodified comments, and only the changes
// required to make it Go code.
//
// - Names are uppercased to export them.
//
// - All fields have JSON tags added to correct the names.
//
// - Fields marked with a ? are also marked as "omitempty".
//
// - Fields that are "|| null" are made pointers.
//
// - Fields that are string or number are left as string.
//
// - Fields that are type "number" are made float64.
package protocol
