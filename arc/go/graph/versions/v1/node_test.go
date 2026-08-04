// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/arc/graph/versions/v1"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Node", func() {
	Describe("DecodeMsgpack", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := v1.Node{
				Key:      "node1",
				Position: spatial.XY{X: 100, Y: 200},
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v1.Node
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal("node1"))
			Expect(decoded.Position).To(Equal(spatial.XY{X: 100, Y: 200}))
		})
		It(
			"Should decode legacy uppercase Go field names, dropping inline type and config",
			func() {
				legacy := struct {
					Key      string
					Type     string
					Config   xmsgpack.EncodedJSON
					Position spatial.XY
				}{
					Key:      "node1",
					Type:     "fn1",
					Config:   xmsgpack.EncodedJSON{"gain": 1},
					Position: spatial.XY{X: 50, Y: 75},
				}
				data := MustSucceed(msgpack.Marshal(legacy))
				var decoded v1.Node
				Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
				Expect(decoded.Key).To(Equal("node1"))
				Expect(decoded.Position).To(Equal(spatial.XY{X: 50, Y: 75}))
			},
		)
	})
})
