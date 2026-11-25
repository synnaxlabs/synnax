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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v2 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v2"
	"github.com/synnaxlabs/x/encoding/orc"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

var _ = Describe("Description breaker", func() {
	It("Should return an error, not panic, on a record from before the field", func() {
		w := orc.NewWriter(0)
		Expect(v2.Channel{
			Name:        "pressure",
			Leaseholder: node.Key(1),
			DataType:    telem.DataType("float32"),
			LocalKey:    2,
		}.EncodeOrc(w)).To(Succeed())
		empty := orc.NewWriter(0)
		empty.String("")
		full := w.Bytes()
		old := full[:len(full)-len(empty.Bytes())]
		r := orc.NewReader(nil)
		r.ResetBytes(old)
		var decoded v2.Channel
		Expect(func() { _ = decoded.DecodeOrc(r) }).ToNot(Panic())
		r.ResetBytes(old)
		Expect(decoded.DecodeOrc(r)).ToNot(Succeed())
	})
})
