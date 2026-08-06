// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/legacy"
)

var _ = Describe("Legacy", func() {
	It("Should report the Console's only written version as LastVersion", func() {
		Expect(legacy.LastVersion).To(BeEquivalentTo(1))
	})

	It("Should decode a Console symbol file through the re-exported Data", func() {
		var d legacy.Data
		Expect(json.Unmarshal([]byte(`{
			"version": 1,
			"name": "Valve",
			"data": {
				"svg": "<svg/>",
				"variant": "valve",
				"handles": [{"key": "h1", "orientation": "left"}],
				"states": [{
					"key": "s1",
					"regions": [{"key": "r1", "selectors": ["#body"]}]
				}]
			}
		}`), &d)).To(Succeed())
		Expect(d.Spec).To(BeAssignableToTypeOf(legacy.Spec{}))
		Expect(d.Spec.SVG).To(Equal("<svg/>"))
		Expect(d.Spec.Handles).To(ConsistOf(BeAssignableToTypeOf(legacy.Handle{})))
		Expect(d.Spec.States).To(ConsistOf(BeAssignableToTypeOf(legacy.State{})))
		Expect(d.Spec.States[0].Regions).
			To(ConsistOf(BeAssignableToTypeOf(legacy.Region{})))
	})
})
