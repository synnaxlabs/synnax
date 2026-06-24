// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package good

// Local stubs stand in for the dot-imported Ginkgo/testutil symbols; the analyzer
// matches by name, so real imports are unnecessary.
func RunSpecs(...any) bool                       { return true }
func BeforeSuite(...any) bool                    { return true }
func BeforeAll(...any) bool                      { return true }
func ShouldNotLeakGoroutines(...any)             {}
func ShouldNotLeakGoroutinesPerSpec(...any) bool { return true }
func openFixture()                               {}

var _ = ShouldNotLeakGoroutinesPerSpec()

func TestGood() { RunSpecs() }

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	openFixture()
})

var _ = BeforeAll(func() {
	ShouldNotLeakGoroutines()
	openFixture()
})
