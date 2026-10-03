// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package license_test

import (
	"crypto/mldsa"
	"encoding/base64"
	"os"
	"strings"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	. "github.com/synnaxlabs/x/testutil"
)

// The portal's contract spec signs testdata/portal.lic with its real claim builder
// and regenerates both fixtures.
var _ = Describe("Portal license key", func() {
	var (
		anchors license.Anchors
		key     string
	)

	BeforeEach(func() {
		raw := MustSucceed(os.ReadFile("testdata/portal.pub"))
		pub := MustSucceed(base64.StdEncoding.DecodeString(
			strings.TrimSpace(string(raw)),
		))
		anchors = license.Anchors{
			"contract": MustSucceed(mldsa.NewPublicKey(mldsa.MLDSA44(), pub)),
		}
		key = strings.TrimSpace(string(MustSucceed(os.ReadFile(
			"testdata/portal.lic",
		))))
	})

	It("Should decode every claim the portal signs", func() {
		issued := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
		expires := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
		Expect(license.Verify(anchors, key)).To(Equal(license.License{
			Jti:               uuid.MustParse("0f2c6a7e-6b1a-4c1d-9c3e-1f1f0b7a2d11"),
			Iat:               uint32(issued.Unix()),
			Exp:               new(uint32(expires.Unix())),
			ClaimsVersion:     1,
			Organization:      uuid.MustParse("64156293-6534-416c-99d1-8db73a22ca6a"),
			Edition:           license.EditionEnterprise,
			Fingerprints:      []string{strings.Repeat("a", 64)},
			FingerprintScheme: 1,
			Machines:          2,
			Channels:          50,
			MaxVersion:        new("0.62"),
			Required:          []string{},
		}))
	})

	It("Should refuse the key once its payload is altered", func() {
		parts := strings.Split(key, ".")
		payload := MustSucceed(base64.RawURLEncoding.DecodeString(parts[1]))
		altered := strings.Replace(string(payload), `"machines":2`, `"machines":9`, 1)
		Expect(altered).ToNot(Equal(string(payload)))
		parts[1] = base64.RawURLEncoding.EncodeToString([]byte(altered))
		Expect(license.Verify(anchors, strings.Join(parts, "."))).Error().
			To(MatchError(license.ErrInvalid))
	})

	It("Should refuse the key under an unknown key id", func() {
		Expect(license.Verify(license.Anchors{}, key)).Error().
			To(MatchError(ContainSubstring(`unknown key "contract"`)))
	})
})
