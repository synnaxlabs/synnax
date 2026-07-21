// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
)

var _ = Describe("Symbol", func() {
	Describe("GorpKey", func() {
		It("Should return the symbol's key", func() {
			k := uuid.New()
			Expect(symbol.Symbol{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(symbol.Symbol{}.SetOptions()).To(BeNil())
		})
	})
	Describe("OntologyID", func() {
		It("Should return the symbol ontology identifier", func() {
			k := uuid.New()
			Expect(symbol.Symbol{Key: k}.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeSchematicSymbol, Key: k.String(),
			}))
		})
	})
})
