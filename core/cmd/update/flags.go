// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package update

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/synnax/pkg/storage"
)

func addFlags(cmd *cobra.Command) {
	instrumentation.AddFlags(cmd)
	start.AddDataFlag(cmd)
}

func buildBackupsConfig(ins alamos.Instrumentation) storage.BackupsConfig {
	return storage.BackupsConfig{
		Instrumentation: ins,
		Dirname:         viper.GetString(start.FlagData),
	}
}
