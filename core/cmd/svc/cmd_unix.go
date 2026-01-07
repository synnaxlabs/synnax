// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package svc

import "github.com/spf13/cobra"

// RegisterCommands is a no-op on non-Windows platforms. Service commands are only
// available on Windows.
func RegisterCommands(*cobra.Command) error { return nil }
