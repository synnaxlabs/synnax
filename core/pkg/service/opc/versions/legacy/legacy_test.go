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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/opc/versions/legacy"
	tasklegacy "github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// The read and write stores wire no rewrite, so their released camelCase shape
// converts through the bare era pass; these specs pin the OPC spellings against it.
var _ = Describe("Era normalization", func() {
	It("Should convert the released read channel shape", func() {
		in := msgpack.EncodedJSON{
			"dataSaving": true,
			"channels": []any{map[string]any{
				"nodeId":     "NS=2;S=fixture.node",
				"nodeName":   "Fixture Node",
				"dataType":   "float32",
				"useAsIndex": false,
				"enabled":    false,
			}},
		}
		Expect(tasklegacy.Rewrite{}.Apply(in)).To(Equal(msgpack.EncodedJSON{
			"data_saving_disabled": false,
			"channels": []any{map[string]any{
				"node_id":      "NS=2;S=fixture.node",
				"node_name":    "Fixture Node",
				"data_type":    "float32",
				"use_as_index": false,
				"disabled":     true,
			}},
		}))
	})

	It("Should convert the released write channel shape", func() {
		in := msgpack.EncodedJSON{
			"channels": []any{map[string]any{
				"nodeId":     "NS=3;I=42",
				"cmdChannel": float64(101),
				"dataType":   "int32",
			}},
		}
		Expect(tasklegacy.Rewrite{}.Apply(in)).To(Equal(msgpack.EncodedJSON{
			"channels": []any{map[string]any{
				"node_id":     "NS=3;I=42",
				"cmd_channel": float64(101),
				"data_type":   "int32",
			}},
		}))
	})
})

var _ = Describe("Scan", func() {
	It("Should rename the driver scan fields", func() {
		in := msgpack.EncodedJSON{"scan_rate": 0.5, "enabled": false}
		Expect(legacy.Scan.Apply(in)).To(Equal(msgpack.EncodedJSON{
			"rate":     0.5,
			"disabled": true,
		}))
	})
})
