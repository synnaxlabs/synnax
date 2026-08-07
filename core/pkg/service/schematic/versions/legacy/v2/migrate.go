// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v1"

// Migrate transforms v1 schematic data into v2 by restamping the version. The
// fields added at v2 are UI-only and are not modeled on the wire here.
func Migrate(old v1.Data) Data {
	d := Data(old)
	d.Version = Version
	return d
}
