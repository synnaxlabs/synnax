// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

// CustomTypeName gives ReadConfig records the table prefix of v4, so the v5 migration
// rewrites each stored config in place.
func (ReadConfig) CustomTypeName() string { return "http_read_config" }
