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
	"github.com/synnaxlabs/alamos"
	cmdcert "github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/listener"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("GetCoreConfigFromViper", func() {
	BeforeEach(func() {
		viper.Reset()
		viper.SetDefault(listener.FlagListen, "localhost:9090")
	})

	It("Should build the config from a scalar listen address", func() {
		viper.Set(listener.FlagListen, "localhost:9091")
		viper.Set(FlagInsecure, true)
		viper.Set(FlagData, "/mnt/ssd")
		viper.Set(FlagPeers, []string{"core02:9090"})
		cfg := MustSucceed(GetCoreConfigFromViper(alamos.Instrumentation{}))
		Expect(cfg.listeners).To(HaveLen(1))
		Expect(cfg.listeners.AdvertiseAddress()).To(Equal(address.Address("localhost:9091")))
		Expect(cfg.certFactoryConfig.Hosts).To(ConsistOf(address.Address("localhost:9091")))
		Expect(*cfg.insecure).To(BeTrue())
		Expect(cfg.dataPath).To(Equal("/mnt/ssd"))
		Expect(cfg.peers).To(ConsistOf(address.Address("core02:9090")))
	})

	It("Should reject a listen list combined with a global cert flag", func() {
		viper.Set(cmdcert.FlagAutoCert, true)
		viper.Set(listener.FlagListen, []any{
			map[string]any{"address": "core01:9090", "cert": map[string]any{"source": "auto"}},
		})
		Expect(GetCoreConfigFromViper(alamos.Instrumentation{})).
			Error().To(MatchError(ContainSubstring("cannot be combined with a listen list")))
	})
})
