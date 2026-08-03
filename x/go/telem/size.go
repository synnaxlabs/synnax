// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import "github.com/synnaxlabs/x/telem/versions"

const (
	// Byte is a single byte.
	Byte Size = versions.Byte
	// Kilobyte is 1,000 bytes.
	Kilobyte Size = versions.Kilobyte
	// Megabyte is 1,000 kilobytes.
	Megabyte Size = versions.Megabyte
	// Gigabyte is 1,000 megabytes.
	Gigabyte Size = versions.Gigabyte
	// Terabyte is 1,000 gigabytes.
	Terabyte Size = versions.Terabyte
	// Petabyte is 1,000 terabytes.
	Petabyte Size = versions.Petabyte
	// Exabyte is 1,000 petabytes.
	Exabyte Size = versions.Exabyte
)
