// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"
)

var errServiceNotSupported = errors.New("Windows Service commands are only supported on Windows")

func serviceInstall(_ *cobra.Command, _ []string) error {
	return errServiceNotSupported
}

func serviceUninstall(_ *cobra.Command, _ []string) error {
	return errServiceNotSupported
}

func serviceStart(_ *cobra.Command, _ []string) error {
	return errServiceNotSupported
}

func serviceStop(_ *cobra.Command, _ []string) error {
	return errServiceNotSupported
}
