// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolution

import "github.com/synnaxlabs/oracle/parser"

// TypeDef represents a top-level type definition from an Oracle schema.
// Unlike an alias (type A = B), a type definition creates a distinct named type
// (type A B in Go, NewType in Python, etc.).
//
// Example schema:
//
//	Key uint32
//	Key uint32 { @go output "core/pkg/service/rack" }
//	DeviceKey rack.Key
type TypeDef struct {
	AST           parser.ITypeDefDefContext
	Name          string
	Namespace     string
	FilePath      string
	QualifiedName string
	// BaseType is the underlying type (primitive or another TypeDef).
	BaseType *TypeRef
	Domains  map[string]Domain
}
