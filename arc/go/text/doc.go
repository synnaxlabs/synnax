// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text

import (
	"github.com/synnaxlabs/arc/text/types/v1"
	"github.com/synnaxlabs/x/crdt"
)

// SeedReplica is the replica id the server uses when seeding or materializing a
// document. The server never authors characters of its own (clients do), so the replica
// only needs to be stable and distinct from the zero root sentinel; clients must choose
// a different replica.
const SeedReplica = v1.SeedReplica

// Create builds a Document from raw source text, attributing every character to the
// seed replica. It initializes the replicated document for an arc created or imported
// with plain text.
func Create(raw string) Document {
	doc := crdt.New(SeedReplica)
	doc.Insert(0, raw)
	inserts, deletes := doc.Snapshot()
	return Document{Inserts: inserts, Deletes: deletes}
}
