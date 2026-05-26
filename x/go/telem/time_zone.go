// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import "github.com/synnaxlabs/x/errors"

// Validate returns an error if tz is not one of the defined TimeZone constants.
func (tz TimeZone) Validate() error {
	switch tz {
	case TimeZoneLocal, TimeZoneUTC:
		return nil
	default:
		return errors.Newf("invalid time zone %q", tz)
	}
}
