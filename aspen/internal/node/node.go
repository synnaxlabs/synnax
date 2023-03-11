// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import (
	"strconv"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/version"
)

type Key uint32

func (i Key) Parse(str string) (Key, error) {
	key, err := strconv.Atoi(str)
	return Key(key), err
}

type Node struct {
	Key        Key
	Address   address.Address
	State     State
	Heartbeat version.Heartbeat
}

func (n Node) Digest() Digest { return Digest{Key: n.Key, Heartbeat: n.Heartbeat} }

type State uint32

const (
	StateHealthy State = iota
	StateSuspect
	StateDead
	StateLeft
)

type Digest struct {
	Key       Key
	Heartbeat version.Heartbeat
}

type Digests map[Key]Digest
