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
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/flags"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/x/address"
)

func configureStartFlags() {
	flags.ConfigureServer(startCmd)
}

func parseIntegrationsFlag() []string {
	enabled := viper.GetStringSlice(flags.EnableIntegrations)
	disabled := viper.GetStringSlice(flags.DisableIntegrations)
	if len(enabled) > 0 {
		return enabled
	}
	return lo.Filter(driver.AllIntegrations, func(integration string, _ int) bool {
		return !lo.Contains(disabled, integration)
	})
}

func parsePeerAddressFlag() []address.Address {
	peerStrings := viper.GetStringSlice(flags.Peers)
	peerAddresses := make([]address.Address, len(peerStrings))
	for i, listenString := range peerStrings {
		peerAddresses[i] = address.Address(listenString)
	}
	return peerAddresses
}
