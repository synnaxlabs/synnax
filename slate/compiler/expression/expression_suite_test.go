// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	"encoding/hex"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/compiler/expression"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	. "github.com/synnaxlabs/x/testutil"
)

func compileExpression(source string) ([]byte, string) {
	expr := MustSucceed(parser.ParseExpression(source))
	module := wasm.NewModule()
	symbols := &symbol.Scope{}
	ctx := compiler.NewContext(module, symbols)
	ctx.EnterFunction("test", nil)
	compiler := expression.NewCompiler(ctx)
	exprType := MustSucceed(compiler.Compile(expr))
	return compiler.Bytes(), exprType.String()
}

func hexToBytes(hexStr string) []byte {
	cleanHex := strings.ReplaceAll(hexStr, " ", "")
	bytes := MustSucceed(hex.DecodeString(cleanHex))
	return bytes
}

func TestExpression(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expression Suite")
}
