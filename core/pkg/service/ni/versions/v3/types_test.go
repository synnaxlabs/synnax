// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"io"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
	"github.com/synnaxlabs/x/encoding/orc"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

var _ = Describe("ScanConfig", func() {
	Describe("ApplyDefaults", func() {
		It("Should fill the default scan rate", func() {
			var cfg v3.ScanConfig
			cfg.ApplyDefaults()
			Expect(cfg.Rate).To(Equal(telem.Rate(0.2)))
		})
	})

	Describe("DecodeOrc", func() {
		DescribeTable(
			"Should fail on a truncated config",
			func(n int) {
				w := orc.NewWriter(0)
				Expect(v3.ScanConfig{
					Key:      uuid.New(),
					Rate:     telem.Rate(5),
					Disabled: true,
				}.EncodeOrc(w)).To(Succeed())
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes()[:n])
				var decoded v3.ScanConfig
				Expect(decoded.DecodeOrc(r)).To(MatchError(io.EOF))
			},
			Entry("missing key", 0),
			Entry("missing rate", 16),
			Entry("missing disabled", 24),
		)
	})
})
