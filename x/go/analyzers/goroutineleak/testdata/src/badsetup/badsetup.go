// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package badsetup

func BeforeSuite(...any) bool                    { return true }
func BeforeAll(...any) bool                      { return true }
func BeforeEach(...any) bool                     { return true }
func ShouldNotLeakGoroutines(...any)             {}
func ShouldNotLeakGoroutinesPerSpec(...any) bool { return true }
func openFixture()                               {}

var _ = ShouldNotLeakGoroutinesPerSpec()

// Missing the leak check entirely.
var _ = BeforeSuite(func() { // want "BeforeSuite must call ShouldNotLeakGoroutines"
	openFixture()
})

// Present, but not the first statement.
var _ = BeforeAll(func() { // want "BeforeAll must call ShouldNotLeakGoroutines"
	openFixture()
	ShouldNotLeakGoroutines() // want "may only be called as the first statement"
})

// Correct: leak check is the first statement — no diagnostic.
var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	openFixture()
})

// BeforeEach is a per-spec node, covered by ShouldNotLeakGoroutinesPerSpec — not flagged.
var _ = BeforeEach(func() {
	openFixture()
})
