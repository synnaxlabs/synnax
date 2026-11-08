// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// ResourceIDs returns the resource.ID for each key.
func ResourceIDs(ks channel.Keys) []resource.ID {
	return lo.Map(ks, func(key channel.Key, _ int) resource.ID {
		return resource.ID{Type: resource.TypeFramer, Key: key.String()}
	})
}
