// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v4"

// Migrate transforms v4 schematic data into v5 by restamping the version. The
// fields added at v5 are UI-only and are not modeled on the wire here.
func Migrate(old v4.Data) Data {
	d := Data(old)
	d.Version = Version
	return d
}
