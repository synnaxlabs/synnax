// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

// mapFields returns "file:struct.field" for every struct field in the generated
// types under root whose type contains a map.
func mapFields(root string) []string {
	GinkgoHelper()
	var found []string
	Expect(filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".gen.go") {
			return nil
		}
		f, err := parser.ParseFile(
			token.NewFileSet(),
			path,
			nil,
			parser.SkipObjectResolution,
		)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range st.Fields.List {
				hasMap := false
				ast.Inspect(field.Type, func(tn ast.Node) bool {
					if _, isMap := tn.(*ast.MapType); isMap {
						hasMap = true
					}
					return !hasMap
				})
				if !hasMap {
					continue
				}
				name := "<embedded>"
				if len(field.Names) > 0 {
					name = field.Names[0].Name
				}
				found = append(
					found, path+":"+ts.Name.Name+"."+name,
				)
			}
			return true
		})
		return nil
	})).To(Succeed())
	return found
}

// Task config records must never store a map whose keys are data: legacy import
// converts unknown camelCase keys to snake_case, so a user-defined key (a header
// name, an enum label) stored in key position gets silently rewritten. Data-bearing
// pairs belong in a list of structs, like http.Header.
var _ = Describe("Config record shapes", func() {
	DescribeTable("holds no map-typed fields in any generated config type",
		func(root string) {
			Expect(MustSucceed(filepath.Glob(root))).ToNot(BeEmpty())
			Expect(mapFields(root)).To(BeEmpty())
		},
		EntryDescription("%s"),
		Entry(nil, "../ethercat/versions"),
		Entry(nil, "../http/versions"),
		Entry(nil, "../labjack/versions"),
		Entry(nil, "../modbus/versions"),
		Entry(nil, "../ni/versions"),
		Entry(nil, "../opc/versions"),
		Entry(nil, "../pagerduty/versions"),
		Entry(nil, "../arc/task/versions"),
		Entry(nil, "../rack/task/versions"),
	)
})
