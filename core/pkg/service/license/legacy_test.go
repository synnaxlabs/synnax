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
	"fmt"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	. "github.com/synnaxlabs/x/testutil"
)

// numericKey returns a key in the unsigned numeric format that expires on the local
// date of exp and covers the given channel count.
func numericKey(exp time.Time, channels int) string {
	exp = exp.Local()
	return fmt.Sprintf(
		"%02d%02d%02d-%08d-0400500003",
		(exp.Year()-2000+89)%100,
		(int(exp.Month())+43)%100,
		(exp.Day()+77)%100,
		(channels+64317284)%100000000,
	)
}

var _ = Describe("ParseLegacy", func() {
	expiry := time.Date(2027, 3, 14, 0, 0, 0, 0, time.Local)
	withChecksum := func(sum string) string {
		return numericKey(expiry, 250)[:16] + sum
	}

	It("should read the expiry and the channel count", func() {
		lic := MustSucceed(license.ParseLegacy(numericKey(expiry, 250)))
		Expect(lic.Exp).To(HaveValue(BeEquivalentTo(expiry.Unix())))
		Expect(lic.Channels).To(BeEquivalentTo(250))
		Expect(lic.Edition).To(Equal(license.EditionEnterprise))
		Expect(lic.Fingerprints).To(BeEmpty())
		Expect(lic.MaxVersion).To(BeNil())
	})
	It("should give the same key the same identifier", func() {
		first := MustSucceed(license.ParseLegacy(numericKey(expiry, 250)))
		second := MustSucceed(license.ParseLegacy(numericKey(expiry, 250)))
		other := MustSucceed(license.ParseLegacy(numericKey(expiry, 251)))
		Expect(first.Jti).To(Equal(second.Jti))
		Expect(first.Jti).ToNot(Equal(other.Jti))
		Expect(first.Jti).ToNot(Equal(uuid.Nil()))
	})
	DescribeTable(
		"Refused keys",
		func(key string) {
			Expect(license.ParseLegacy(key)).Error().To(MatchError(license.ErrInvalid))
		},
		Entry("a signed key", "a.b.c"),
		Entry("too few digits", "12345-12345678-0400500003"),
		Entry("a date that does not exist", "166320"+numericKey(expiry, 250)[6:]),
		Entry("no channels", numericKey(expiry, 0)),
		Entry("a second checksum digit that is not 4", withChecksum("0900000003")),
		Entry(
			"leading checksum digits not a multiple of 9",
			withChecksum("0400600003"),
		),
		Entry("middle checksum digits not a multiple of 7", withChecksum("0400510003")),
		Entry("a last checksum digit under 3", withChecksum("0400500002")),
		Entry("a last checksum digit over 6", withChecksum("0400500007")),
	)
})
