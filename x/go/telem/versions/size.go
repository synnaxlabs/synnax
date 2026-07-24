// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import v0 "github.com/synnaxlabs/x/telem/versions/v0"

const (
	// Byte is a single byte.
	Byte Size = v0.Byte
	// Kilobyte is 1,000 bytes.
	Kilobyte Size = v0.Kilobyte
	// Megabyte is 1,000 kilobytes.
	Megabyte Size = v0.Megabyte
	// Gigabyte is 1,000 megabytes.
	Gigabyte Size = v0.Gigabyte
	// Terabyte is 1,000 gigabytes.
	Terabyte Size = v0.Terabyte
	// Petabyte is 1,000 terabytes.
	Petabyte Size = v0.Petabyte
	// Exabyte is 1,000 petabytes.
	Exabyte Size = v0.Exabyte
)
