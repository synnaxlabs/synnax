// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package relapse provides type-safe reducer patterns with code generation
// for discriminated union actions.
//
// # Usage
//
// Define payload structs with a Handle method that mutates state:
//
//	type AddItem struct {
//		Item string `json:"item"`
//	}
//
//	func (a AddItem) Handle(state State) (State, error) {
//		state.Items = append(state.Items, a.Item)
//		return state, nil
//	}
//
// Then add a go:generate directive to your package:
//
//	//go:generate go run github.com/synnaxlabs/x/relapse/gen -state State
//
// Run `go generate` to create a reducer.go file with:
//   - Action constants (ActionAddItem = "add_item")
//   - Discriminated union Action struct
//   - Reduce(state State, action Action) function
//   - Constructor functions (NewAddItemAction)
package relapse
