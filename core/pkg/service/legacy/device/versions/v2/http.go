// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"slices"

	"github.com/synnaxlabs/gorp/migrate"
)

// HTTPBlankHeaderMigration drops each health check header with an empty name or value
// from a stored HTTP device, because the Driver refuses one.
var HTTPBlankHeaderMigration = migrate.NewEntry(
	"v63_http_health_check_blank_headers",
	func(_ context.Context, d Device) (Device, error) {
		if d.Make != "http" {
			return d, nil
		}
		hc, ok := d.Properties["health_check"].(map[string]any)
		if !ok {
			return d, nil
		}
		headers, ok := hc["headers"].([]any)
		if !ok {
			return d, nil
		}
		hc["headers"] = slices.DeleteFunc(headers, func(h any) bool {
			header, ok := h.(map[string]any)
			return ok && (header["name"] == "" || header["value"] == "")
		})
		return d, nil
	},
)
