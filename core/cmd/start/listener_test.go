// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package start

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var factoryCfg = cert.FactoryConfig{LoaderConfig: cert.DefaultLoaderConfig}

func validateSecure(listeners []listenerSpec) error {
	v := validate.New("test")
	validateListeners(v, listeners, false)
	return v.Error()
}

func listenerObj(address string, source string) map[string]any {
	return map[string]any{
		"address": address,
		"cert":    map[string]any{"source": source},
	}
}

var _ = Describe("Listener", func() {
	BeforeEach(func() {
		viper.Reset()
		viper.SetDefault(FlagListen, "localhost:9090")
	})

	Describe("parseListeners", func() {
		It("Should parse a scalar address into a single file-source listener", func() {
			viper.Set(FlagListen, "localhost:9091")
			specs := MustSucceed(parseListeners(factoryCfg))
			Expect(specs).To(HaveLen(1))
			Expect(specs[0].address).To(Equal(address.Address("localhost:9091")))
			Expect(specs[0].advertise).To(BeTrue())
			Expect(specs[0].cert.source).To(Equal(cert.SourceTypeFile))
			Expect(specs[0].cert.cert).To(Equal(factoryCfg.AbsoluteNodeCertPath()))
			Expect(specs[0].cert.key).To(Equal(factoryCfg.AbsoluteNodeKeyPath()))
		})

		It("Should parse a list of listener objects", func() {
			viper.Set(FlagListen, []any{
				map[string]any{
					"address":   "core01:9090",
					"cert":      map[string]any{"source": "file", "cert": "d.crt", "key": "d.key"},
					"advertise": true,
				},
				listenerObj("node01:9091", "tailscale"),
			})
			specs := MustSucceed(parseListeners(factoryCfg))
			Expect(specs).To(HaveLen(2))
			Expect(specs[0].address).To(Equal(address.Address("core01:9090")))
			Expect(specs[0].cert.cert).To(Equal("d.crt"))
			Expect(specs[0].advertise).To(BeTrue())
			Expect(specs[1].cert.source).To(Equal(cert.SourceTypeTailscale))
			Expect(specs[1].advertise).To(BeFalse())
		})

		It("Should reject a list combined with --auto-cert", func() {
			viper.Set(FlagAutoCert, true)
			viper.Set(FlagListen, []any{listenerObj("core01:9090", "auto")})
			_, err := parseListeners(factoryCfg)
			Expect(err).To(MatchError(ContainSubstring("cannot be combined with a listen list")))
		})
	})

	Describe("validateListeners", func() {
		It("Should accept a valid secure configuration", func() {
			Expect(validateSecure([]listenerSpec{
				{address: "a:9090", cert: certSpec{source: cert.SourceTypeAuto}},
				{address: "b:9091", cert: certSpec{source: cert.SourceTypeFile, cert: "c", key: "k"}},
			})).To(Succeed())
		})

		It("Should reject more than one advertised listener", func() {
			Expect(validateSecure([]listenerSpec{
				{address: "a:9090", cert: certSpec{source: cert.SourceTypeAuto}, advertise: true},
				{address: "b:9091", cert: certSpec{source: cert.SourceTypeAuto}, advertise: true},
			})).To(MatchError(ContainSubstring("at most one listener may advertise")))
		})

		It("Should reject duplicate addresses", func() {
			Expect(validateSecure([]listenerSpec{
				{address: "a:9090", cert: certSpec{source: cert.SourceTypeAuto}},
				{address: "a:9090", cert: certSpec{source: cert.SourceTypeAuto}},
			})).To(MatchError(ContainSubstring("duplicate listener address")))
		})

		It("Should require both cert and key for a file source", func() {
			Expect(validateSecure([]listenerSpec{
				{address: "a:9090", cert: certSpec{source: cert.SourceTypeFile, cert: "c"}},
			})).To(MatchError(ContainSubstring("file source requires both cert and key")))
		})

		It("Should reject cert or key on an auto source", func() {
			Expect(validateSecure([]listenerSpec{
				{address: "a:9090", cert: certSpec{source: cert.SourceTypeAuto, cert: "c"}},
			})).To(MatchError(ContainSubstring("must not set cert or key")))
		})

		It("Should require a source in secure mode", func() {
			Expect(validateSecure([]listenerSpec{
				{address: "a:9090"},
			})).To(MatchError(ContainSubstring("certificate source is required")))
		})

		It("Should skip source checks in insecure mode", func() {
			v := validate.New("test")
			validateListeners(v, []listenerSpec{{address: "a:9090"}}, true)
			Expect(v.Error()).To(Succeed())
		})
	})

	Describe("advertiseAddress", func() {
		It("Should return the first listener when none advertise", func() {
			c := CoreConfig{listeners: []listenerSpec{{address: "a:1"}, {address: "b:2"}}}
			Expect(c.advertiseAddress()).To(Equal(address.Address("a:1")))
		})

		It("Should return the advertised listener", func() {
			c := CoreConfig{listeners: []listenerSpec{
				{address: "a:1"},
				{address: "b:2", advertise: true},
			}}
			Expect(c.advertiseAddress()).To(Equal(address.Address("b:2")))
		})
	})
})
