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
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/cmd/listener"
	"github.com/synnaxlabs/synnax/cmd/start"
)

var _ = Describe("Removed name validation flag breaker", func() {
	const env = "SYNNAX_DISABLE_CHANNEL_NAME_VALIDATION"
	removed := ContainSubstring("--disable-channel-name-validation was removed")
	BeforeEach(func() {
		viper.Reset()
		viper.SetDefault(listener.FlagListen, "localhost:9090")
		viper.Set(start.FlagInsecure, true)
		viper.SetEnvPrefix("synnax")
		viper.AutomaticEnv()
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		DeferCleanup(viper.Reset)
	})

	DescribeTable("Should fail the start for a true environment value",
		func(value string) {
			GinkgoT().Setenv(env, value)
			Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).Error().
				To(MatchError(removed))
		},
		Entry("true", "true"),
		Entry("1", "1"),
		Entry("t", "t"),
		Entry("TRUE", "TRUE"),
		Entry("True", "True"),
	)

	DescribeTable("Should start for a false or empty environment value",
		func(value string) {
			GinkgoT().Setenv(env, value)
			Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).Error().
				ToNot(HaveOccurred())
		},
		Entry("false", "false"),
		Entry("0", "0"),
		Entry("empty", ""),
	)

	It("Should start with the variable unset", func() {
		GinkgoT().Setenv(env, "x")
		Expect(os.Unsetenv(env)).To(Succeed())
		Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).Error().
			ToNot(HaveOccurred())
	})

	DescribeTable("Should fail the start for a true flag value",
		func(args ...string) {
			cmd := start.NewCmd()
			Expect(cmd.ParseFlags(args)).To(Succeed())
			Expect(cmd.PreRunE(cmd, nil)).To(Succeed())
			Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).Error().
				To(MatchError(removed))
		},
		Entry("bare", "--disable-channel-name-validation"),
		Entry("=true", "--disable-channel-name-validation=true"),
		Entry("=1", "--disable-channel-name-validation=1"),
		Entry("=t", "--disable-channel-name-validation=t"),
		Entry("=TRUE", "--disable-channel-name-validation=TRUE"),
	)

	It("Should start for a false flag value", func() {
		cmd := start.NewCmd()
		Expect(cmd.ParseFlags([]string{"--disable-channel-name-validation=false"})).
			To(Succeed())
		Expect(cmd.PreRunE(cmd, nil)).To(Succeed())
		Expect(start.GetCoreConfigFromViper(alamos.Instrumentation{})).Error().
			ToNot(HaveOccurred())
	})

	It("Should keep the flag out of the help", func() {
		cmd := start.NewCmd()
		Expect(
			cmd.Flags().Lookup("disable-channel-name-validation").Hidden,
		).To(BeTrue())
	})
})
