/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package rack

import "github.com/synnaxlabs/synnax/pkg/distribution/core"

type Key uint32

func NewKey(node core.NodeKey, localKey uint16) Key {
	return Key(uint32(node)<<16 | uint32(localKey))
}

func (k Key) Node() core.NodeKey { return core.NodeKey(k >> 16) }

func (k Key) LocalKey() uint16 { return uint16(uint32(k) & 0xFFFF) }

type Rack struct {
	Name     string
	Node     core.NodeKey
	LocalKey uint16
}
