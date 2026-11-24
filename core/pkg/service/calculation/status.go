// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"fmt"

	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
)

// StatusFromError builds the error status for the calculation with key, whose output
// is named name, when its expression fails to compile.
func StatusFromError(key Key, name string, err error) *Status {
	return &Status{
		Owner:       new(ResourceID(key)),
		Name:        name,
		Variant:     status.VariantError,
		Message:     "Failed to compile calculation",
		Description: err.Error(),
		Time:        telem.Now(),
	}
}

// primaryChangedStatus builds the status that the primary of the calculation with key,
// whose output is named name, moved from the channel named from to the one named to. It
// has no owner, so the graph's status of the calculation stays.
func primaryChangedStatus(key Key, name, from, to string) *Status {
	return &Status{
		Key:     primaryStatusKey(key),
		Name:    name,
		Variant: status.VariantInfo,
		Message: "Changed the primary input",
		Description: fmt.Sprintf(
			"The expression no longer reads %s, so the outputs now use the timestamps "+
				"of %s.",
			from,
			to,
		),
		Time: telem.Now(),
	}
}
