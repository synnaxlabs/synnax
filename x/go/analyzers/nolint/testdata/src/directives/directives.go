// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package directives

//nolint:all // want "fix the finding"
func all() {}

func inline() int {
	x := 1 //nolint:ineffassign // want "fix the finding"
	x = 2  //nolint // want "fix the finding"
	// nolint:unused // want "fix the finding"
	return x
}
