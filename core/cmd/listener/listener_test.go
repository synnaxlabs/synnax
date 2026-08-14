// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package listener_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	cmdcert "github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/listener"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/auto"
	"github.com/synnaxlabs/synnax/pkg/security/cert/file"
	"github.com/synnaxlabs/synnax/pkg/security/cert/tailscale"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/x/address"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

func listenerObj(address, source string) map[string]any {
	return map[string]any{
		"address": address,
		"cert":    map[string]any{"source": source},
	}
}

var _ = Describe("Listener", func() {
	BeforeEach(func() {
		viper.Reset()
		viper.SetDefault(listener.FlagListen, "localhost:9090")
	})

	Describe("Parse", func() {
		It("Should parse a scalar address into a single file-source listener", func() {
			viper.Set(listener.FlagListen, "localhost:9091")
			configs := MustSucceed(listener.Parse())
			Expect(configs).To(HaveLen(1))
			Expect(configs[0].Address).To(Equal(address.Address("localhost:9091")))
			Expect(configs[0].Advertise).To(BeTrue())
			Expect(configs[0].Cert.Source).To(Equal(file.SourceType))
			Expect(configs[0].Cert.Cert).To(BeEmpty())
			Expect(configs[0].Cert.Key).To(BeEmpty())
		})

		It("Should parse a list of listener objects", func() {
			viper.Set(listener.FlagListen, []any{
				map[string]any{
					"address": "core01:9090",
					"cert": map[string]any{
						"source": "file",
						"cert":   "d.crt",
						"key":    "d.key",
					},
					"advertise": true,
				},
				listenerObj("node01:9091", "tailscale"),
			})
			configs := MustSucceed(listener.Parse())
			Expect(configs).To(HaveLen(2))
			Expect(configs[0].Address).To(Equal(address.Address("core01:9090")))
			Expect(configs[0].Cert.Cert).To(Equal("d.crt"))
			Expect(configs[0].Advertise).To(BeTrue())
			Expect(configs[1].Cert.Source).To(Equal(tailscale.SourceType))
			Expect(configs[1].Advertise).To(BeFalse())
		})

		It("Should reject a list combined with --auto-cert", func() {
			viper.Set(cmdcert.FlagAutoCert, true)
			viper.Set(listener.FlagListen, []any{listenerObj("core01:9090", "auto")})
			Expect(listener.Parse()).
				Error().
				To(MatchError(ContainSubstring("cannot be combined with a listen list")))
		})
	})

	Describe("Validate", func() {
		It("Should accept a valid configuration", func() {
			Expect(listener.Configs{
				{Address: "a:9090"},
				{Address: "b:9091"},
			}.Validate()).To(Succeed())
		})

		It("Should require at least one listener", func() {
			Expect(listener.Configs(nil).Validate()).
				To(MatchError(ContainSubstring("at least one listener is required")))
		})

		It("Should reject an empty address", func() {
			Expect(listener.Configs{{Address: ""}}.Validate()).
				To(MatchError(ContainSubstring("address")))
		})

		It("Should reject more than one advertised listener", func() {
			Expect(listener.Configs{
				{Address: "a:9090", Advertise: true},
				{Address: "b:9091", Advertise: true},
			}.Validate()).To(MatchError(ContainSubstring("at most one listener may advertise")))
		})

		It("Should reject duplicate addresses", func() {
			Expect(listener.Configs{
				{Address: "a:9090"},
				{Address: "a:9090"},
			}.Validate()).To(MatchError(ContainSubstring("duplicate listener address")))
		})

		DescribeTable("Should reject a cert or key path on a source that ignores them",
			func(source, certPath, keyPath string) {
				Expect(listener.Configs{
					{
						Address: "a:9090",
						Cert: listener.CertConfig{
							Source: source,
							Cert:   certPath,
							Key:    keyPath,
						},
					},
				}.Validate()).To(MatchError(ContainSubstring("does not read a cert or key path")))
			},
			Entry("auto with cert path", auto.SourceType, "n.crt", ""),
			Entry("auto with key path", auto.SourceType, "", "n.key"),
			Entry("tailscale with cert path", tailscale.SourceType, "n.crt", ""),
		)

		It("Should accept a cert and key path on the file source", func() {
			Expect(listener.Configs{
				{
					Address: "a:9090",
					Cert: listener.CertConfig{
						Source: file.SourceType,
						Cert:   "n.crt",
						Key:    "n.key",
					},
				},
			}.Validate()).To(Succeed())
		})
	})

	Describe("ValidateAdvertiseSource", func() {
		It("Should reject a tailscale source on the advertised listener", func() {
			Expect(listener.Configs{
				{
					Address:   "a:9090",
					Cert:      listener.CertConfig{Source: tailscale.SourceType},
					Advertise: true,
				},
				{Address: "b:9091", Cert: listener.CertConfig{Source: auto.SourceType}},
			}.ValidateAdvertiseSource()).
				To(MatchError(ContainSubstring("advertised listener cannot use the Tailscale source")))
		})

		It("Should reject a tailscale source on the implicit first listener", func() {
			Expect(listener.Configs{
				{
					Address: "a:9090",
					Cert:    listener.CertConfig{Source: tailscale.SourceType},
				},
				{Address: "b:9091", Cert: listener.CertConfig{Source: auto.SourceType}},
			}.ValidateAdvertiseSource()).
				To(MatchError(ContainSubstring("advertised listener cannot use the Tailscale source")))
		})

		It("Should accept a tailscale source on a non-advertised listener", func() {
			Expect(listener.Configs{
				{
					Address:   "a:9090",
					Cert:      listener.CertConfig{Source: auto.SourceType},
					Advertise: true,
				},
				{
					Address: "b:9091",
					Cert:    listener.CertConfig{Source: tailscale.SourceType},
				},
			}.ValidateAdvertiseSource()).To(Succeed())
		})
	})

	Describe("AdvertiseAddress", func() {
		It("Should return the first listener when none advertise", func() {
			Expect(listener.Configs{
				{Address: "a:1"},
				{Address: "b:2"},
			}.AdvertiseAddress()).To(Equal(address.Address("a:1")))
		})

		It("Should return the advertised listener", func() {
			Expect(listener.Configs{
				{Address: "a:1"},
				{Address: "b:2", Advertise: true},
			}.AdvertiseAddress()).To(Equal(address.Address("b:2")))
		})
	})

	Describe("Resolve", func() {
		It("Should reject an unknown certificate source", func() {
			fs := xfs.NewMem()
			mock.GenerateCerts(fs)
			prov := MustSucceed(security.NewProvider(security.ProviderConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      mock.SmallKeySize,
				Insecure:     new(false),
			}))
			fc := cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      mock.SmallKeySize,
			}
			Expect(listener.Configs{
				{Address: "localhost:9090", Cert: listener.CertConfig{Source: "bogus"}},
			}.Resolve(prov, fc, false)).
				Error().To(MatchError(ContainSubstring("unknown certificate source")))
		})
	})
})
