// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
)

// withLists gives b the lists of a compile that reads reads and writes writes, as each
// stored Arc holds after a Core start.
func withLists(b *arc.Base, reads, writes []channel.Key) {
	b.Reads, b.Writes = reads, writes
}
