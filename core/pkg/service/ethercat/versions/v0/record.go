// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

// CustomTypeName gives ReadConfig records a table prefix that cannot collide with the
// identically named config types of other integrations.
func (ReadConfig) CustomTypeName() string { return "ethercat_read_config" }

// CustomTypeName gives WriteConfig records a table prefix that cannot collide with the
// identically named config types of other integrations.
func (WriteConfig) CustomTypeName() string { return "ethercat_write_config" }

// CustomTypeName gives ScanConfig records a table prefix that cannot collide with the
// identically named config types of other integrations.
func (ScanConfig) CustomTypeName() string { return "ethercat_scan_config" }
