// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cli

// Flag constants for type safety
const (
	verboseFlag = "verbose"
	schemasFlag = "schemas"
)

func configureRootFlags() {
	rootCmd.PersistentFlags().BoolP(
		verboseFlag,
		"v",
		false,
		"Enable verbose output",
	)
	rootCmd.PersistentFlags().StringSliceP(
		schemasFlag,
		"s",
		nil,
		"Schema file glob patterns (e.g., 'schemas/*.oracle')",
	)
}
