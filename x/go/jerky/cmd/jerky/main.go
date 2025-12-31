// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package main provides the jerky CLI for code generation.
//
// Usage:
//
//	//go:generate jerky
//	type MyStruct struct { ... }
//
// Then run: go generate ./...
package main

func main() {
	Execute()
}
