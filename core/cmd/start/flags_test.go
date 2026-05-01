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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/cmd/start"
)

func TestStart(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Start Suite")
}

var _ = Describe("Flags", func() {
	It("Should register every Go flag-name constant on the command", func() {
		cmd := &cobra.Command{Use: "test"}
		start.AddFlags(cmd)

		expected := []string{
			start.FlagListen,
			start.FlagPeers,
			start.FlagData,
			start.FlagMem,
			start.FlagInsecure,
			start.FlagUsername,
			start.FlagPassword,
			start.FlagAutoCert,
			start.FlagNoDriver,
			start.FlagSlowConsumerTimeout,
			start.FlagEnableIntegrations,
			start.FlagDisableIntegrations,
			start.FlagTaskOpTimeout,
			start.FlagTaskPollInterval,
			start.FlagTaskShutdownTimeout,
			start.FlagTaskWorkerCount,
			start.FlagDisableChannelNameValidation,
			cert.FlagCertsDir,
			cert.FlagCAKey,
			cert.FlagCACert,
			cert.FlagNodeKey,
			cert.FlagNodeCert,
			cert.FlagAllowKeyReuse,
			cert.FlagKeySize,
			instrumentation.FlagVerbose,
			instrumentation.FlagDebug,
			instrumentation.FlagLogFilePath,
			instrumentation.FlagLogFileMaxSize,
			instrumentation.FlagLogFileMaxBackups,
			instrumentation.FlagLogFileMaxAge,
			instrumentation.FlagLogFileCompress,
		}
		for _, name := range expected {
			Expect(cmd.Flags().Lookup(name)).
				ToNot(BeNil(), "missing flag %q", name)
		}
	})
})
