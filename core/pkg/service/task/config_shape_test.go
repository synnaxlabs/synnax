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
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

// configVersionDirs returns the versions directory of every package under root that
// owns task config stores. A package qualifies by declaring a Stores() []config.Store
// method, the same method layer.go concatenates to assemble the config registry, so a
// new integration is covered as soon as it is wired.
func configVersionDirs(root string) []string {
	GinkgoHelper()
	var found []string
	Expect(filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "versions" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
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
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name.Name != "Stores" {
				continue
			}
			if !returnsConfigStores(fn.Type.Results) {
				continue
			}
			found = append(found, filepath.Join(filepath.Dir(path), "versions"))
		}
		return nil
	})).To(Succeed())
	slices.Sort(found)
	return slices.Compact(found)
}

// returnsConfigStores reports whether results is a single []config.Store.
func returnsConfigStores(results *ast.FieldList) bool {
	if results == nil || len(results.List) != 1 {
		return false
	}
	arr, ok := results.List[0].Type.(*ast.ArrayType)
	if !ok {
		return false
	}
	sel, ok := arr.Elt.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Store" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "config"
}

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
	It("holds no map-typed fields in any generated config type", func() {
		dirs := configVersionDirs("..")
		Expect(dirs).ToNot(BeEmpty())
		var found []string
		for _, dir := range dirs {
			Expect(MustSucceed(filepath.Glob(dir))).ToNot(BeEmpty(), dir)
			found = append(found, mapFields(dir)...)
		}
		Expect(found).To(BeEmpty())
	})
})
