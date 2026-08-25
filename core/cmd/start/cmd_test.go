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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	cmdcert "github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/listener"
	"github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	. "github.com/synnaxlabs/x/testutil"
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
		"Should allow a Tailscale advertised listener when no peers are configured",
		func() {
			viper.Set(listener.FlagListen, []any{
				map[string]any{
					"address": "node01.example-tailnet.ts.net:9090",
					"cert":    map[string]any{"source": "tailscale"},
				},
			})
			cfg := MustSucceed(start.GetCoreConfigFromViper(alamos.Instrumentation{}))
			Expect(cfg.Validate()).ToNot(MatchError(
				ContainSubstring("advertised listener cannot use the Tailscale source"),
			))
		},
	)
})
