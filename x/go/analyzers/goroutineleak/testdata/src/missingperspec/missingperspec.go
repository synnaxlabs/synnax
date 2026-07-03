// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package missingperspec

func RunSpecs(...any) bool                       { return true }
func ShouldNotLeakGoroutinesPerSpec(...any) bool { return true }

// No `var _ = ShouldNotLeakGoroutinesPerSpec()` at package scope, so the suite is
// flagged at its RunSpecs call.
func TestMissing() {
	RunSpecs() // want "does not register ShouldNotLeakGoroutinesPerSpec"
}
