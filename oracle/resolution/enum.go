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

// Enum represents a resolved enum definition from an Oracle schema.
type Enum struct {
	AST           parser.IEnumDefContext
	Name          string
	Namespace     string
	FilePath      string
	QualifiedName string
	Values        []EnumEntry
	IsIntEnum     bool
	Domains       map[string]Domain
}

// EnumEntry represents a single value within an enum.
type EnumEntry struct {
	Name string
	ExpressionValue
}
