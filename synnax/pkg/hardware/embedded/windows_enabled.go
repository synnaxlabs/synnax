// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build driver && windows

package embedded

import (
	"embed"
	"os/exec"
)

//go:embed assets/driver.exe
//go:embed assets/driver_without_ni.exe
var executable embed.FS

// driverPath is the path to the driver executable
const driverName = "driver.exe"
const driverWithoutNIName = "driver_without_ni.exe"

func configureSysProcAttr(cmd *exec.Cmd) {
	return
}
