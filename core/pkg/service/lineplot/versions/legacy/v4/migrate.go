// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import v3 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v3"

// Migrate transforms v3 line plot data into v4 by rewriting the version string. The
// UI-only fields added at v4 (measure, annotations) are not modeled on the wire here
// since they do not survive the final lift.
func Migrate(old v3.Data) Data {
	d := Data(old)
	d.Version = Version
	return d
}
