// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package start_test

import (
	"io/fs"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	cmdcert "github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/listener"
	"github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("GetCoreConfigFromViper", func() {
	BeforeEach(func() {
		viper.Reset()
		viper.SetDefault(listener.FlagListen, "localhost:9090")
	})

	It("Should build a config from a scalar listen address", func() {
		viper.Set(listener.FlagListen, "localhost:9091")
		viper.Set(start.FlagInsecure, true)
		Expect(
			start.GetCoreConfigFromViper(alamos.Instrumentation{}),
		).Error().
			ToNot(HaveOccurred())
	})

	It("Should reject a listen list combined with a global cert flag", func() {
		viper.Set(cmdcert.FlagAutoCert, true)
		viper.Set(listener.FlagListen, []any{
			map[string]any{
				"address": "core01:9090",
				"cert":    map[string]any{"source": "auto"},
			},
		})
		Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).
			Error().
			To(MatchError(ContainSubstring("cannot be combined with a listen list")))
	})

	It(
		"Should accept a listen list when the node cert flags hold their defaults",
		func() {
			viper.SetDefault(
				cmdcert.FlagNodeCert,
				cert.DefaultLoaderConfig.NodeCertPath,
			)
			viper.SetDefault(cmdcert.FlagNodeKey, cert.DefaultLoaderConfig.NodeKeyPath)
			viper.Set(listener.FlagListen, []any{
				map[string]any{
					"address": "core01:9090",
					"cert":    map[string]any{"source": "auto"},
				},
			})
			Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).
				Error().
				ToNot(HaveOccurred())
		},
	)

	It(
		"Should reject a Tailscale advertised listener when peers are configured",
		func() {
			viper.Set(listener.FlagListen, []any{
				map[string]any{
					"address": "node01.example-tailnet.ts.net:9090",
					"cert":    map[string]any{"source": "tailscale"},
				},
			})
			viper.Set(start.FlagPeers, []string{"core02:9090"})
			cfg := MustSucceed(start.GetCoreConfigFromViper(alamos.Instrumentation{}))
			Expect(cfg.Validate()).To(MatchError(
				ContainSubstring("advertised listener cannot use the Tailscale source"),
			))
		},
	)

	It(
		"Should reject a Tailscale advertised listener when no peers are configured",
		func() {
			viper.Set(listener.FlagListen, []any{
				map[string]any{
					"address": "node01.example-tailnet.ts.net:9090",
					"cert":    map[string]any{"source": "tailscale"},
				},
			})
			cfg := MustSucceed(start.GetCoreConfigFromViper(alamos.Instrumentation{}))
			Expect(cfg.Validate()).To(MatchError(
				ContainSubstring("advertised listener cannot use the Tailscale source"),
			))
		},
	)

	Describe("Session idle timeout", func() {
		It("Should fill the default when the flag is unset", func() {
			viper.Set(start.FlagInsecure, true)
			cfg := MustSucceed(start.GetCoreConfigFromViper(alamos.Instrumentation{}))
			Expect(start.DefaultCoreConfig.Override(cfg).Validate()).
				ToNot(MatchError(ContainSubstring("session_idle_timeout")))
		})

		It("Should reject a negative timeout", func() {
			viper.Set(start.FlagInsecure, true)
			viper.Set(start.FlagSessionIdleTimeout, -time.Second)
			cfg := MustSucceed(start.GetCoreConfigFromViper(alamos.Instrumentation{}))
			Expect(start.DefaultCoreConfig.Override(cfg).Validate()).
				To(MatchError(ContainSubstring("session_idle_timeout")))
		})
	})

	Describe("Auth methods", func() {
		validate := func() error {
			GinkgoHelper()
			viper.Set(start.FlagInsecure, true)
			cfg := MustSucceed(start.GetCoreConfigFromViper(alamos.Instrumentation{}))
			return start.DefaultCoreConfig.Override(cfg).Validate()
		}

		It("Should accept every method when the flag is unset", func() {
			Expect(validate()).ToNot(MatchError(ContainSubstring("auth_methods")))
		})

		It("Should reject an explicitly empty list", func() {
			viper.Set(start.FlagAuthMethods, []string{})
			Expect(validate()).To(MatchError(ContainSubstring("auth_methods")))
		})

		It("Should reject an unknown method", func() {
			viper.Set(start.FlagAuthMethods, []string{"password", "carrier_pigeon"})
			Expect(validate()).To(MatchError(ContainSubstring("carrier_pigeon")))
		})

		It("Should refuse api_key off while the embedded Driver is on", func() {
			viper.Set(start.FlagAuthMethods, []string{"password"})
			Expect(validate()).To(
				MatchError(ContainSubstring("must include api_key")),
			)
		})

		It("Should allow password off while the embedded Driver is on", func() {
			viper.Set(start.FlagAuthMethods, []string{"api_key"})
			Expect(validate()).ToNot(MatchError(ContainSubstring("auth_methods")))
		})

		It("Should allow api_key off when the embedded Driver is off", func() {
			viper.Set(start.FlagAuthMethods, []string{"password"})
			viper.Set(start.FlagNoDriver, true)
			Expect(validate()).ToNot(MatchError(ContainSubstring("auth_methods")))
		})
	})

	Describe("OIDC providers", func() {
		It("Should accept a list of providers", func() {
			viper.Set(start.FlagOIDCProviders, []any{map[string]any{
				"name":                   "idp",
				"issuer":                 "https://idp.example.com",
				"client-id":              "synnax",
				"client-secret":          "secret",
				"user-creation-disabled": true,
			}})
			MustSucceed(start.GetCoreConfigFromViper(alamos.Instrumentation{}))
		})

		It("Should reject a value that is not a list", func() {
			viper.Set(start.FlagOIDCProviders, "idp")
			Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).Error().
				To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("oidc-providers must be a list")),
				))
		})

		It("Should reject an entry that is not an object", func() {
			viper.Set(start.FlagOIDCProviders, []any{"idp"})
			Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).Error().
				To(MatchError(ContainSubstring("oidc-providers[0] must be an object")))
		})

		It("Should reject a setting of the wrong type", func() {
			viper.Set(start.FlagOIDCProviders, []any{map[string]any{
				"name":                   "idp",
				"user-creation-disabled": "true",
			}})
			Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).Error().
				To(MatchError(ContainSubstring(
					"oidc-providers[0].user-creation-disabled must be a bool, not string",
				)))
		})

		It("Should reject an unknown setting", func() {
			viper.Set(start.FlagOIDCProviders, []any{map[string]any{"secret": "x"}})
			Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).Error().
				To(MatchError(ContainSubstring("oidc-providers[0].secret is not a setting")))
		})
	})

	It("Should allow a Tailscale advertised listener in insecure mode", func() {
		viper.Set(listener.FlagListen, []any{
			map[string]any{
				"address": "node01.example-tailnet.ts.net:9090",
				"cert":    map[string]any{"source": "tailscale"},
			},
		})
		viper.Set(start.FlagInsecure, true)
		cfg := MustSucceed(start.GetCoreConfigFromViper(alamos.Instrumentation{}))
		Expect(cfg.Validate()).ToNot(MatchError(
			ContainSubstring("advertised listener cannot use the Tailscale source"),
		))
	})

	Describe("License key", func() {
		var missing string
		BeforeEach(func() {
			missing = filepath.Join(GinkgoT().TempDir(), "missing.jwt")
			DeferCleanup(func() {
				viper.Set(start.FlagLicenseKey, "")
				viper.Set(start.FlagLicenseFile, "")
			})
		})

		It("Should name the flag when the license file cannot be read", func() {
			viper.Set(start.FlagLicenseFile, missing)
			Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).
				Error().
				To(And(
					MatchError(fs.ErrNotExist),
					MatchError(ContainSubstring(
						"failed to read the --license-file file",
					)),
				))
		})

		It("Should not read the license file when the key flag is set", func() {
			viper.Set(start.FlagLicenseKey, "key")
			viper.Set(start.FlagLicenseFile, missing)
			MustSucceed(start.GetCoreConfigFromViper(alamos.Instrumentation{}))
		})

		It("Should fall back to the license file on a blank key flag", func() {
			viper.Set(start.FlagLicenseKey, " \n")
			viper.Set(start.FlagLicenseFile, missing)
			Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).
				Error().
				To(MatchError(fs.ErrNotExist))
		})
	})
})

var _ = Describe("DriverTrustAnchors", func() {
	It("Should return nil in insecure mode", func() {
		Expect(MustSucceed(
			start.DriverTrustAnchors(true, cert.LoaderConfig{}),
		)).To(BeNil())
	})

	It("Should return the CA and node certificates", func() {
		fs := xfs.NewMem()
		mock.GenerateCerts(fs)
		Expect(MustSucceed(
			start.DriverTrustAnchors(false, cert.LoaderConfig{FS: fs}),
		)).ToNot(BeEmpty())
	})

	It("Should fail when no certificates exist", func() {
		Expect(start.DriverTrustAnchors(false, cert.LoaderConfig{FS: xfs.NewMem()})).
			Error().
			To(MatchError(validate.ErrValidation))
	})
})
