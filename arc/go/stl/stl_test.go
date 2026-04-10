// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stl_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

func collectModuleResolvers(r symbol.Resolver) []*symbol.ModuleResolver {
	switch v := r.(type) {
	case *symbol.ModuleResolver:
		return []*symbol.ModuleResolver{v}
	case symbol.CompoundResolver:
		var all []*symbol.ModuleResolver
		for _, sub := range v {
			all = append(all, collectModuleResolvers(sub)...)
		}
		return all
	default:
		return nil
	}
}

var _ = Describe("SymbolResolver", func() {
	It("Should set KindFunction on every module member with a function type", func() {
		var violations []string
		for _, mod := range collectModuleResolvers(stl.SymbolResolver) {
			for name, sym := range mod.Members {
				if sym.Type.Kind != types.KindFunction {
					continue
				}
				if sym.Kind != symbol.KindFunction {
					violations = append(violations, fmt.Sprintf(
						"%s.%s (Kind is %s, expected KindFunction)", mod.Name, name, sym.Kind,
					))
				}
			}
		}
		Expect(violations).To(BeEmpty(),
			"Module members with function types missing Kind: symbol.KindFunction:\n  "+
				strings.Join(violations, "\n  "))
	})
})
