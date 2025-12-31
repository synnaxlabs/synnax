// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package jerky provides code generation that derives protobuf definitions and
// translation functions from Go struct definitions.
//
// Jerky treats Go structs as the single source of truth for data structures,
// automatically generating:
//
//  1. Protobuf message definitions (.proto files)
//  2. Forward/backward translation functions between Go and protobuf types
//  3. Version-aware migration infrastructure for schema evolution
//  4. MarshalGorp/UnmarshalGorp methods for gorp integration
//
// Usage:
//
// Add the following directive to a struct:
//
//	//go:generate jerky
//	type MyStruct struct {
//	    Key      uuid.UUID `json:"key"`
//	    Name     string    `json:"name"`
//	    Duration int       `json:"duration"`
//	}
//
// Then run `go generate ./...` to generate the code.
package jerky

import (
	"github.com/synnaxlabs/x/jerky/parse"
)

// Version is the current version of jerky.
const Version = "1.0.0"

// ParsedStruct is an alias to parse.ParsedStruct for external use.
type ParsedStruct = parse.ParsedStruct

// ParsedField is an alias to parse.ParsedField for external use.
type ParsedField = parse.ParsedField
