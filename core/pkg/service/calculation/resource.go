// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import "github.com/synnaxlabs/synnax/pkg/service/resource"

// ResourceID returns the resource ID of the calculation with key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeCalculation, Key: k.String()}
}
