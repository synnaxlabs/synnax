// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import "github.com/synnaxlabs/synnax/pkg/service/node"

// NewKey instantiates a new rack key from its node and local key components.
func NewKey(node node.Key, localKey uint16) Key {
	return Key(uint32(node)<<16 | uint32(localKey))
}

func StatusKey(k Key) string { return k.OntologyID().String() }
