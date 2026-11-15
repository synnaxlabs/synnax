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
	"encoding/base64"
	"io/fs"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	cmdcert "github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/listener"
	"github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/errors"
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
		"Should reject a Tailscale advertised listener when the Driver is on",
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
					MatchError(
						func(err error) bool {
							return errors.Is(err, validate.ErrValidation)
						},
						"validate.ErrValidation",
					),
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

	Describe("Secrets key", func() {
		BeforeEach(func() {
			DeferCleanup(func() { viper.Set(start.FlagSecretsKey, "") })
		})

		It("Should accept a base64 key", func() {
			key := secret.NewEncryptionKey()
			viper.Set(start.FlagSecretsKey, base64.StdEncoding.EncodeToString(key[:]))
			MustSucceed(start.GetCoreConfigFromViper(alamos.Instrumentation{}))
		})

		It("Should name the flag when the key is not valid", func() {
			viper.Set(start.FlagSecretsKey, "short")
			Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).
				Error().
				To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"invalid --secrets-key: secrets key is not base64",
					)),
				))
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
		testutil.GenerateCerts(fs)
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

var _ = Describe("AddDataFlag", func() {
	It("Should add the data flag with its shorthand and default", func() {
		cmd := &cobra.Command{}
		start.AddDataFlag(cmd)
		flag := cmd.Flags().Lookup(start.FlagData)
		Expect(flag).ToNot(BeNil())
		Expect(flag.Shorthand).To(Equal("d"))
		Expect(flag.DefValue).To(Equal("synnax-data"))
	})
})
