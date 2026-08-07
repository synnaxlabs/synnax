// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/v2"
)

// withProps builds a v1.Data carrying props for a single node.
func withProps(props map[string]any) v1.Data {
	return v1.Data{
		Graph: v1.Graph{Props: map[string]map[string]any{"n1": props}},
		Mode:  "graph",
	}
}

var _ = Describe("Migrate", func() {
	It("Should remap set_status props onto status.set", func() {
		out := v2.Migrate(withProps(map[string]any{
			"key":       "set_status",
			"statusKey": "pump_state",
			"variant":   "error",
			"message":   "pump down",
		}))

		Expect(out.Graph.Props["n1"]).To(Equal(map[string]any{
			"key":         "status.set",
			"key_or_name": "pump_state",
			"variant":     "error",
			"message":     "pump down",
		}))
	})

	It("Should default the variant and fill absent fields empty", func() {
		out := v2.Migrate(withProps(map[string]any{"key": "set_status"}))

		Expect(out.Graph.Props["n1"]).To(Equal(map[string]any{
			"key":         "status.set",
			"key_or_name": "",
			"variant":     "success",
			"message":     "",
		}))
	})

	It("Should leave props for every other symbol untouched", func() {
		props := map[string]any{"key": "stl.on", "channel": "valve_cmd"}

		Expect(v2.Migrate(withProps(props)).Graph.Props["n1"]).To(Equal(props))
	})
})
