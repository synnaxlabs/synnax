// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Doc comments", func() {
	It("Should keep the doc comment of Dispatch on Dispatch", func() {
		f := MustSucceed(parser.ParseFile(
			token.NewFileSet(),
			"channel.go",
			nil,
			parser.ParseComments,
		))
		docs := map[string]string{}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok {
				docs[fn.Name.Name] = fn.Doc.Text()
			}
		}
		Expect(docs["Dispatch"]).To(HavePrefix("Dispatch "))
		Expect(strings.Contains(docs["editedInputs"], "Dispatch applies")).To(BeFalse())
	})
})
