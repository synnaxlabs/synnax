// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package missing

func RunSpecs(...any) bool { return true }
func SuiteConfig() int     { return 0 }

func TestMissing() {
	RunSpecs("Missing Suite") // want "RunSpecs must pass SuiteConfig\\(\\)"
}

func TestUncalled() {
	RunSpecs("Uncalled Suite", SuiteConfig) // want "RunSpecs must pass SuiteConfig\\(\\)"
}
