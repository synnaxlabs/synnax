// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v10_test

import (
	"fmt"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
)

var _ = Describe("Schematic", func() {
	Describe("GorpKey", func() {
		It("Should return the schematic's key", func() {
			k := uuid.New()
			Expect(v10.Schematic{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v10.Schematic{}.SetOptions()).To(BeNil())
		})
	})
	Describe("ResourceID", func() {
		It("Should return the schematic resource identifier", func() {
			k := uuid.New()
			Expect(v10.Schematic{Key: k}.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeSchematic, Key: k.String(),
			}))
		})
	})

	Describe("Validate", func() {
		newSchematic := func(align v10.FlexAlignment) v10.Schematic {
			configs := make(map[string]v10.ElementConfig, 340)
			for i := range 340 {
				configs[fmt.Sprintf("element-%d", i)] = v10.ElementConfig{
					Variant: v10.ValueElementConfig{},
				}
			}
			configs["element-170"] = v10.ElementConfig{
				Variant: v10.TextBoxElementConfig{Align: align},
			}
			s := v10.Schematic{Name: "schematic", Configs: configs}
			s.ApplyDefaults()
			return s
		}

		It("Should accept many valid configs", func() {
			Expect(newSchematic("start").Validate()).To(Succeed())
		})

		It("Should format the error of one invalid config among many", func() {
			// Map order sets how many valid configs follow the invalid one.
			for range 5 {
				s := newSchematic("left")
				formatted := make(chan string, 1)
				go func() {
					defer GinkgoRecover()
					formatted <- s.Validate().Error()
				}()
				Eventually(formatted, 5*time.Second).Should(
					Receive(ContainSubstring("invalid align: left")),
				)
			}
		})
	})
})
