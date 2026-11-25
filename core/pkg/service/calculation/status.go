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

// StaleStatus builds the warning that the join of the calculation whose output is named
// name gave no output for some samples of its primary input.
func StaleStatus(name string) Status {
	return Status{
		Name:    name,
		Variant: status.VariantWarning,
		Message: "Inputs went stale",
		Description: "Some samples of the primary input got no output, because another " +
			"input had no sample within its hold limit. Check that each input is still " +
			"written.",
	}
}

// PrimaryChangedStatus builds the status that the primary of the calculation with key,
// whose output is named name, moved from the channel named from, which its expression
// no longer reads, to the one named to. It has no owner, so the calculation's own
// status stays.
func PrimaryChangedStatus(key Key, name, from, to string) *Status {
	return primaryStatus(key, name, fmt.Sprintf(
		"The expression no longer reads %s, so the outputs now use the timestamps of %s.",
		from,
		to,
	))
}

// PrimaryStoredStatus builds the status that the primary of the calculation with key,
// whose output is named name, moved from the virtual channel named from to the stored
// channel named to. It has no owner, so the calculation's own status stays.
func PrimaryStoredStatus(key Key, name, from, to string) *Status {
	return primaryStatus(key, name, fmt.Sprintf(
		"%s is virtual, so the outputs now use the timestamps of %s.",
		from,
		to,
	))
}

func primaryStatus(key Key, name, description string) *Status {
	return &Status{
		Key:         PrimaryStatusKey(key),
		Name:        name,
		Variant:     status.VariantInfo,
		Message:     "Changed the primary input",
		Description: description,
		Time:        telem.Now(),
	}
}
