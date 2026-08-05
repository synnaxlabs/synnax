// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package strays

// Local stubs stand in for the dot-imported Ginkgo/testutil symbols; the analyzer
// matches by name, so real imports are unnecessary.
func RunSpecs(...any) bool                       { return true }
func BeforeSuite(...any) bool                    { return true }
func BeforeEach(...any) bool                     { return true }
func Describe(...any) bool                       { return true }
func It(...any) bool                             { return true }
func ShouldNotLeakGoroutines(...any)             {}
func ShouldNotLeakGoroutinesPerSpec(...any) bool { return true }
func openFixture()                               {}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = ShouldNotLeakGoroutinesPerSpec() // want "already registered in this package"

func TestStrays() {
	RunSpecs()
	ShouldNotLeakGoroutinesPerSpec() // want "may only be registered at package scope"
}

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	openFixture()
})

var _ = BeforeEach(func() {
	ShouldNotLeakGoroutines() // want "first statement of a BeforeSuite or BeforeAll"
	openFixture()
})

var _ = It("leaks", func() {
	openFixture()
	ShouldNotLeakGoroutines() // want "first statement of a BeforeSuite or BeforeAll"
})

var _ = Describe("nested", func() {
	ShouldNotLeakGoroutinesPerSpec() // want "may only be registered at package scope"
})

var _ = BeforeSuite(namedSetup)

func namedSetup() {
	ShouldNotLeakGoroutines()
	openFixture()
}
