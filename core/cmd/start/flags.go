// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package start

import (
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding/base64"
)

const (
	FlagListen              = "listen"
	FlagPeers               = "peers"
	FlagData                = "data"
	FlagMem                 = "mem"
	FlagInsecure            = "insecure"
	FlagUsername            = "username"
	FlagPassword            = "password"
	FlagAutoCert            = "auto-cert"
	FlagNoDriver            = "no-driver"
	FlagSlowConsumerTimeout = "slow-consumer-timeout"
	FlagEnableIntegrations  = "enable-integrations"
	FlagDisableIntegrations = "disable-integrations"
	FlagTaskOpTimeout       = "task-op-timeout"
	FlagTaskPollInterval    = "task-poll-interval"
	FlagTaskShutdownTimeout = "task-shutdown-timeout"
	FlagTaskWorkerCount     = "task-worker-count"
)

// BindFlags binds the start flags to the given command.
func BindFlags(cmd *cobra.Command) {
	cert.BindFlags(cmd)
	cmd.Flags().StringP(
		FlagListen,
		"l",
		"localhost:9090",
		`The address to listen for client connections.`,
	)
	cmd.Flags().StringSliceP(
		FlagPeers,
		"p",
		nil,
		"Addresses of additional peers in the cluster.",
	)
	cmd.Flags().StringSlice(
		FlagEnableIntegrations,
		nil,
		"Device integrations to enable (labjack, modbus, ni, opc, sequence)",
	)
	cmd.Flags().StringSlice(
		FlagDisableIntegrations,
		nil,
		"Device integrations to disable (labjack, modbus, ni, opc, sequence)",
	)
	cmd.Flags().StringP(
		FlagData,
		"d",
		"synnax-data",
		"Directory where the synnax Core will store its data.",
	)
	cmd.Flags().BoolP(FlagMem, "m", false, "Use in-memory storage")
	cmd.Flags().BoolP(
		FlagInsecure,
		"i",
		false,
		"Disable encryption, authentication, and authorization.",
	)
	cmd.Flags().String(FlagUsername, "synnax", "Username for the admin user.")
	cmd.Flags().String(FlagPassword, "seldon", "Password for the admin user.")
	cmd.Flags().Bool(
		FlagAutoCert,
		false,
		"Automatically generate self-signed certificates.",
	)
	cmd.Flags().Bool(FlagNoDriver, false, "Disable the embedded Driver")
	cmd.Flags().Duration(
		FlagSlowConsumerTimeout,
		2500*time.Millisecond,
		"Terminate slow consumers of the relay after this timeout.",
	)
	cmd.Flags().Duration(
		FlagTaskOpTimeout,
		60*time.Second,
		"Duration before reporting stuck task operations in the embedded Driver.",
	)
	cmd.Flags().Duration(
		FlagTaskPollInterval,
		1*time.Second,
		"Interval between task timeout checks in the embedded Driver.",
	)
	cmd.Flags().Duration(
		FlagTaskShutdownTimeout,
		30*time.Second,
		"Max time to wait for task workers during embedded Driver shutdown.",
	)
	cmd.Flags().Int(
		FlagTaskWorkerCount,
		4,
		"Number of worker threads for task operations in the embedded Driver (1-64).",
	)
	cmd.Flags().String(decodedName, "", decodedUsage)
}

var (
	decodedName  = base64.MustDecode("bGljZW5zZS1rZXk=")
	decodedUsage = base64.MustDecode(
		"TGljZW5zZSBrZXkgaW4gZm9ybSAiIyMjIyMjLSMjIyMjIyMjLSMjIyMjIyMjIyMiLg==",
	)
)

func parseIntegrationsFlag() []string {
	enabled := viper.GetStringSlice(FlagEnableIntegrations)
	disabled := viper.GetStringSlice(FlagDisableIntegrations)
	if len(enabled) > 0 {
		return enabled
	}
	return lo.Filter(driver.AllIntegrations, func(integration string, _ int) bool {
		return !lo.Contains(disabled, integration)
	})
}

func parsePeerAddressFlag() []address.Address {
	peers := viper.GetStringSlice(FlagPeers)
	return lo.Map(peers, func(peer string, _ int) address.Address {
		return address.Address(peer)
	})
}
