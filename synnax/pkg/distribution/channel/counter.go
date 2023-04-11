// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"strconv"

	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/x/counter"
	"github.com/synnaxlabs/x/kv"
)

const counterKey = ".distribution.channel.counter"

type keyCounter struct {
	err      error
	internal *kv.PersistedCounter
}

func (c *keyCounter) Add(delta ...uint16) uint16 {
	var total int64
	for _, d := range delta {
		total += int64(d)
	}
	total, c.err = c.internal.Add(total)
	return uint16(total)
}

func (c *keyCounter) Value() uint16 { return uint16(c.internal.Value()) }

func (c *keyCounter) Error() error { return c.err }

func openCounter(nodeKey dcore.NodeKey, kve kv.DB) (counter.Uint16Error, error) {
	c, err := kv.OpenCounter(kve, []byte(strconv.Itoa(int(nodeKey))+counterKey))
	if err != nil {
		return nil, err
	}
	return &keyCounter{internal: c}, nil
}
