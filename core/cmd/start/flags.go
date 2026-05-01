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
	_ "embed"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/flagdef"
	"github.com/synnaxlabs/x/encoding/base64"
)

// Flag names used for starting a Synnax Core.
const (
	FlagListen                       = "listen"
	FlagPeers                        = "peers"
	FlagData                         = "data"
	FlagMem                          = "mem"
	FlagInsecure                     = "insecure"
	FlagUsername                     = "username"
	FlagPassword                     = "password"
	FlagAutoCert                     = "auto-cert"
	FlagNoDriver                     = "no-driver"
	FlagSlowConsumerTimeout          = "slow-consumer-timeout"
	FlagEnableIntegrations           = "enable-integrations"
	FlagDisableIntegrations          = "disable-integrations"
	FlagTaskOpTimeout                = "task-op-timeout"
	FlagTaskPollInterval             = "task-poll-interval"
	FlagTaskShutdownTimeout          = "task-shutdown-timeout"
	FlagTaskWorkerCount              = "task-worker-count"
	FlagDisableChannelNameValidation = "disable-channel-name-validation"
)

//go:embed flags.json
var flagsJSON []byte

// FlagDefs are the parsed flag definitions for the start flag set.
var FlagDefs = flagdef.MustParse(flagsJSON)

// AddFlags adds the start flags to the given command.
func AddFlags(cmd *cobra.Command) {
	cert.AddFlags(cmd)
	flagdef.MustRegister(cmd, FlagDefs)
	cmd.Flags().String(FlagDecoded, "", usage)
}

var (
	FlagDecoded = base64.MustDecode("bGljZW5zZS1rZXk=")
	usage       = base64.MustDecode(
		"TGljZW5zZSBrZXkgaW4gZm9ybSAiIyMjIyMjLSMjIyMjIyMjLSMjIyMjIyMjIyMi",
	)
)
