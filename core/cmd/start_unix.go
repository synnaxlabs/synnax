// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package cmd

import (
	"os"

	"github.com/synnaxlabs/synnax/cmd/exit"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// RunMain is the entry point for the Synnax CLI on non-Windows platforms. It registers
// the UUID encoding, then runs the command, since Windows Service mode does not apply.
func RunMain() {
	msgpack.RegisterUUID()
	if err := NewCmd().Execute(); err != nil {
		os.Exit(exit.Code(err))
	}
}
