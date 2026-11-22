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
