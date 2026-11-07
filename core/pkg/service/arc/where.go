// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
)

// MatchDirectories returns a filter for the Arcs in any of the directories with keys.
func MatchDirectories(keys ...directory.Key) Filter {
	return Match(func(_ gorp.Context, _ Retrieve, a *Arc) (bool, error) {
		return inDirectories(a, keys...), nil
	})
}

func inDirectories(a *Arc, keys ...directory.Key) bool {
	d := a.Base().Directory
	return d != nil && slices.Contains(keys, *d)
}
