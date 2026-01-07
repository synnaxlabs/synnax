// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"encoding/base64"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/x/address"
)

const (
	listenFlag                       = "listen"
	peersFlag                        = "peers"
	dataFlag                         = "data"
	memFlag                          = "mem"
	insecureFlag                     = "insecure"
	usernameFlag                     = "username"
	passwordFlag                     = "password"
	autoCertFlag                     = "auto-cert"
	noDriverFlag                     = "no-driver"
	slowConsumerTimeoutFlag          = "slow-consumer-timeout"
	enableIntegrationsFlag           = "enable-integrations"
	disableIntegrationsFlag          = "disable-integrations"
	taskOpTimeoutFlag                = "task-op-timeout"
	taskPollIntervalFlag             = "task-poll-interval"
	taskShutdownTimeoutFlag          = "task-shutdown-timeout"
	taskWorkerCountFlag              = "task-worker-count"
	disableChannelNameValidationFlag = "disable-channel-name-validation"
)

func configureStartFlags() {
	startCmd.Flags().StringP(
		listenFlag,
		"l",
		"localhost:9090",
		`The address to listen for client connections.`,
	)

	startCmd.Flags().StringSliceP(
		peersFlag,
		"p",
		nil,
		"Addresses of additional peers in the cluster.",
	)

	startCmd.Flags().StringSlice(
		enableIntegrationsFlag,
		nil,
		"Device integrations to enable (labjack, modbus, ni, opc, sequence)",
	)

	startCmd.Flags().StringSlice(
		disableIntegrationsFlag,
		nil,
		"Device integrations to disable (labjack, modbus, ni, opc, sequence)",
	)

	startCmd.Flags().StringP(
		dataFlag,
		"d",
		"synnax-data",
		"ParentDirname where the synnax node will store its data.",
	)

	startCmd.Flags().BoolP(
		memFlag,
		"m",
		false,
		"Use in-memory storage",
	)

	startCmd.Flags().BoolP(
		insecureFlag,
		"i",
		false,
		"Disable encryption, authentication, and authorization.",
	)

	startCmd.Flags().String(
		usernameFlag,
		"synnax",
		"Username for the admin user.",
	)

	startCmd.Flags().String(
		passwordFlag,
		"seldon",
		"Password for the admin user.",
	)

	startCmd.Flags().Bool(
		autoCertFlag,
		false,
		"Automatically generate self-signed certificates.",
	)

	startCmd.Flags().Bool(
		noDriverFlag,
		false,
		"Disable the embedded synnax driver",
	)

	startCmd.Flags().Duration(
		slowConsumerTimeoutFlag,
		2500*time.Millisecond,
		"Terminate slow consumers of the relay after this timeout.",
	)

	decodedName, _ := base64.StdEncoding.DecodeString("bGljZW5zZS1rZXk=")
	decodedUsage, _ := base64.StdEncoding.DecodeString("TGljZW5zZSBrZXkgaW4gZm9ybSAiIyMjIyMjLSMjIyMjIyMjLSMjIyMjIyMjIyMiLg==")

	startCmd.Flags().String(
		string(decodedName),
		"",
		string(decodedUsage),
	)

	startCmd.Flags().Bool(
		disableChannelNameValidationFlag,
		false,
		"Disable channel name validation (allows special characters, spaces, etc.)",
	)
}

func parseIntegrationsFlag() []string {
	enabled := viper.GetStringSlice(enableIntegrationsFlag)
	disabled := viper.GetStringSlice(disableIntegrationsFlag)
	if len(enabled) > 0 {
		return enabled
	}
	return lo.Filter(driver.AllIntegrations, func(integration string, _ int) bool {
		return !lo.Contains(disabled, integration)
	})
}

func parsePeerAddressFlag() []address.Address {
	peerStrings := viper.GetStringSlice(peersFlag)
	peerAddresses := make([]address.Address, len(peerStrings))
	for i, listenString := range peerStrings {
		peerAddresses[i] = address.Address(listenString)
	}
	return peerAddresses
}
