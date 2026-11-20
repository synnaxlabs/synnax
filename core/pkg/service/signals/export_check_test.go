// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// NewReferenceFeedAppend returns the append of a reference feed, as
// PublishReferenceChanges builds it.
func NewReferenceFeedAppend(
	holderOf func(gorp.Target) resource.ID,
) func(gorp.Commit) framer.Frame {
	f := &referenceFeed{
		ch:       channel.Channel{Name: referenceChangeName, DataType: telem.StringT},
		holderOf: holderOf,
		index:    make(map[gorp.Target]int),
		seen:     make(set.Set[targetHolder]),
	}
	return func(c gorp.Commit) framer.Frame { return f.append(c, framer.Frame{}) }
}

// ReferenceRecord returns the record of the reference change channel.
func ReferenceRecord() Record { return Record{Of: targetOfSample, Changed: holdersOfSample} }
