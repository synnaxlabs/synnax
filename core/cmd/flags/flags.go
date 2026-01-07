// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package flags provides shared flag definitions for the Synnax CLI commands.
package flags

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/x/encoding/base64"
)

// Flag name constants for server configuration.
const (
	Listen              = "listen"
	Peers               = "peers"
	Data                = "data"
	Mem                 = "mem"
	Insecure            = "insecure"
	Username            = "username"
	Password            = "password"
	AutoCert            = "auto-cert"
	NoDriver            = "no-driver"
	SlowConsumerTimeout = "slow-consumer-timeout"
	EnableIntegrations  = "enable-integrations"
	DisableIntegrations = "disable-integrations"
	TaskOpTimeout       = "task-op-timeout"
	TaskPollInterval    = "task-poll-interval"
	TaskShutdownTimeout = "task-shutdown-timeout"
	TaskWorkerCount     = "task-worker-count"
)

// ConfigureServer adds the common server configuration flags to the given command. This
// is used by both the start command and the service install command.
func ConfigureServer(cmd *cobra.Command) {
	cmd.Flags().StringP(
		Listen,
		"l",
		"localhost:9090",
		`The address to listen for client connections.`,
	)

	cmd.Flags().StringSliceP(
		Peers,
		"p",
		nil,
		"Addresses of additional peers in the cluster.",
	)

	cmd.Flags().StringSlice(
		EnableIntegrations,
		nil,
		"Device integrations to enable (labjack, modbus, ni, opc, sequence)",
	)

	cmd.Flags().StringSlice(
		DisableIntegrations,
		nil,
		"Device integrations to disable (labjack, modbus, ni, opc, sequence)",
	)

	cmd.Flags().StringP(
		Data,
		"d",
		"synnax-data",
		"Directory where the synnax node will store its data.",
	)

	cmd.Flags().BoolP(Mem, "m", false, "Use in-memory storage")

	cmd.Flags().BoolP(
		Insecure,
		"i",
		false,
		"Disable encryption, authentication, and authorization.",
	)

	cmd.Flags().String(Username, "synnax", "Username for the admin user.")

	cmd.Flags().String(Password, "seldon", "Password for the admin user.")

	cmd.Flags().Bool(
		AutoCert,
		false,
		"Automatically generate self-signed certificates.",
	)

	cmd.Flags().Bool(NoDriver, false, "Disable the embedded synnax Driver")

	cmd.Flags().Duration(
		SlowConsumerTimeout,
		2500*time.Millisecond,
		"Terminate slow consumers of the relay after this timeout.",
	)

	cmd.Flags().Duration(
		TaskOpTimeout,
		60*time.Second,
		"Duration before reporting stuck task operations in the driver.",
	)

	cmd.Flags().Duration(
		TaskPollInterval,
		1*time.Second,
		"Interval between task timeout checks in the driver.",
	)

	cmd.Flags().Duration(
		TaskShutdownTimeout,
		30*time.Second,
		"Max time to wait for task workers during driver shutdown.",
	)

	cmd.Flags().Int(
		TaskWorkerCount,
		4,
		"Number of worker threads for task operations in the driver (1-64).",
	)

	cmd.Flags().String(decodedName, "", decodedUsage)
}

var (
	decodedName  = base64.MustDecode("bGljZW5zZS1rZXk=")
	decodedUsage = base64.MustDecode("TGljZW5zZSBrZXkgaW4gZm9ybSAiIyMjIyMjLSMjIyMjIyMjLSMjIyMjIyMjIyMiLg==")
)
