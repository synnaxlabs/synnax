// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v5 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v5"
	"github.com/synnaxlabs/x/encoding/orc"
	. "github.com/synnaxlabs/x/testutil"
)

//go:embed testdata/*.hex
var wireFixtures embed.FS

var truncatedErr = SatisfyAny(MatchError(io.EOF), MatchError(io.ErrUnexpectedEOF))

// releasedWire returns the frozen Orc payload a released build wrote, stored as hex in
// testdata. It pins the v5 wire format independently of the current codec.
func releasedWire() []byte {
	GinkgoHelper()
	raw := MustSucceed(wireFixtures.ReadFile("testdata/v5_released.hex"))
	return MustSucceed(hex.DecodeString(
		strings.ReplaceAll(string(raw), "\n", ""),
	))
}

var _ = Describe("Released wire format", func() {
	It("Should decode a payload written by a released build", func(ctx SpecContext) {
		var lp v5.LinePlot
		Expect(orc.Codec.Decode(ctx, releasedWire(), &lp)).To(Succeed())
		Expect(lp.Key).To(Equal(uuid.MustParse("251ed216-d959-4684-ac64-a8475c146697")))
		Expect(lp.Name).To(Equal("UT Disk Free"))
	})

	It("Should re-encode the payload byte for byte", func(ctx SpecContext) {
		raw := releasedWire()
		var lp v5.LinePlot
		Expect(orc.Codec.Decode(ctx, raw, &lp)).To(Succeed())
		Expect(orc.Codec.Encode(ctx, lp)).To(Equal(raw))
	})

	// The .decoded.json companion is the fixture's human-readable audit form. Run with
	// UPDATE_DECODED=1 to regenerate it after intentionally replacing the fixture.
	It("Should match the canonical decoded form", func(ctx SpecContext) {
		var lp v5.LinePlot
		Expect(orc.Codec.Decode(ctx, releasedWire(), &lp)).To(Succeed())
		pretty := append(MustSucceed(json.MarshalIndent(lp, "", "  ")), '\n')
		p := filepath.Join("testdata", "v5_released.decoded.json")
		if os.Getenv("UPDATE_DECODED") == "1" {
			Expect(os.WriteFile(p, pretty, 0o644)).To(Succeed())
			return
		}
		Expect(pretty).To(MatchJSON(MustSucceed(os.ReadFile(p))))
	})

	It("Should reject every truncation of the payload", func(ctx SpecContext) {
		raw := releasedWire()
		for n := 3; n < len(raw); n++ {
			var lp v5.LinePlot
			Expect(orc.Codec.Decode(ctx, raw[:n], &lp)).
				To(truncatedErr, "prefix length %d", n)
		}
	})
})
