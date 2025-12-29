// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package generate

import (
	"github.com/synnaxlabs/x/jerky/parse"
	"github.com/synnaxlabs/x/jerky/state"
)

// Export private functions for testing.
var (
	ToSnakeCase      = toSnakeCase
	TypeName         = typeName
	ProtoToGoType    = protoToGoType
	GetVersionFields = getVersionFields
)

// GetProtoType exposes the private getProtoType method for testing.
func (g *Generator) GetProtoType(goType parse.GoType) (string, bool) {
	return g.getProtoType(goType)
}

// GetTranslationExprs exposes the private getTranslationExprs method for testing.
func (g *Generator) GetTranslationExprs(f parse.ParsedField, parentPkg string, parentPath string) (forward, backward string, canFail bool, imports []string, aliasedImports []AliasedImport) {
	return g.getTranslationExprs(f, parentPkg, parentPath)
}

// ComputeMigrationFields exposes the private computeMigrationFields method for testing.
func (g *Generator) ComputeMigrationFields(typeState state.TypeState, fromVH, toVH *state.VersionHistory) []MigrationFieldData {
	return g.computeMigrationFields(typeState, fromVH, toVH)
}

// CollectNestedTypeVersions exposes the private collectNestedTypeVersions method for testing.
func (g *Generator) CollectNestedTypeVersions(parsed parse.ParsedStruct) map[string]state.NestedTypeInfo {
	return g.collectNestedTypeVersions(parsed)
}
