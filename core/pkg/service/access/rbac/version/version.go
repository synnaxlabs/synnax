// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version

type Version = uint8

const (
	// V0 represents unversioned policies from RC (no version field, has Subjects).
	V0 Version = 0
	// V1 represents the first versioned policy format (role-based, no Subjects).
	V1 Version = 1
	// Current is the current policy version.
	Current = V1
)