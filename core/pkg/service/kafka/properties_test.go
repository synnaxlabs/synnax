// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Properties", func() {
	creds := kafka.Credentials{Username: "user", Password: "secret"}

	DescribeTable("Should round trip the SASL mechanism through JSON",
		func(variant kafka.SASLVariant, mechanism string) {
			props := kafka.Properties{
				Brokers: []string{"localhost:9092"},
				Sasl:    kafka.SASL{Variant: variant},
			}
			data := MustSucceed(json.Marshal(props))
			Expect(string(data)).To(ContainSubstring(`"mechanism":"` + mechanism + `"`))
			var decoded kafka.Properties
			Expect(json.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(props))
		},
		Entry("none", kafka.NoneSASL{}, "none"),
		Entry("plain", kafka.PlainSASL{Credentials: creds}, "plain"),
		Entry("scram_sha_256", kafka.ScramSha256SASL{Credentials: creds}, "scram_sha_256"),
		Entry("scram_sha_512", kafka.ScramSha512SASL{Credentials: creds}, "scram_sha_512"),
	)

	It("Should encode an unset mechanism as null", func() {
		data := MustSucceed(json.Marshal(kafka.SASL{}))
		Expect(string(data)).To(Equal("null"))
		var decoded kafka.SASL
		Expect(json.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.Variant).To(BeNil())
	})

	It("Should reject an unknown mechanism", func() {
		var decoded kafka.SASL
		Expect(json.Unmarshal([]byte(`{"mechanism":"kerberos"}`), &decoded)).
			To(MatchError(ContainSubstring(`unknown mechanism "kerberos"`)))
	})

	Describe("ApplyDefaults", func() {
		It("Should default the SASL mechanism to none", func() {
			props := kafka.Properties{Brokers: []string{"localhost:9092"}}
			props.ApplyDefaults()
			Expect(props.Sasl.Variant).To(Equal(kafka.NoneSASL{}))
		})
	})
})
