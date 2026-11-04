// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import "github.com/synnaxlabs/gorp"

// MatchGrouped returns a filter for logs that sit in a group when grouped is true,
// and for logs in no group when grouped is false.
func MatchGrouped(grouped bool) Filter {
	return Match(func(_ gorp.Context, _ Retrieve, e *Log) (bool, error) {
		return (e.Group != nil) == grouped, nil
	})
}
