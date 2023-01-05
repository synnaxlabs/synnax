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
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/version"
	"strconv"
)

type ID uint32

func (i ID) Parse(str string) (ID, error) {
	id, err := strconv.Atoi(str)
	return ID(id), err
}

type Node struct {
	ID        ID
	Address   address.Address
	State     State
	Heartbeat version.Heartbeat
}

func (n Node) Digest() Digest { return Digest{ID: n.ID, Heartbeat: n.Heartbeat} }

type State uint32

const (
	StateHealthy State = iota
	StateSuspect
	StateDead
	StateLeft
)

type Digest struct {
	ID        ID
	Heartbeat version.Heartbeat
}

type Digests map[ID]Digest
