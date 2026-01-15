// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/types"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/version"
)

type Key types.Uint12

const (
	Bootstrapper Key = 1
	Free             = Key(math.MaxUint12)
)

func (k Key) Parse(str string) (Key, error) {
	key, err := strconv.Atoi(str)
	return Key(key), err
}

func (k Key) IsFree() bool { return k == Free }

func (k Key) IsBootstrapper() bool { return k == Bootstrapper }

// String implements fmt.Stringer.
func (k Key) String() string { return strconv.Itoa(int(k)) }

type Node struct {
	Key       Key
	Address   address.Address
	State     State
	Heartbeat version.Heartbeat
}

func (n Node) Digest() Digest { return Digest{Key: n.Key, Heartbeat: n.Heartbeat} }

type Change = change.Change[Key, Node]

type State uint32

func BasicallyEqual(prev, next Node) bool {
	prev.Heartbeat = next.Heartbeat
	return prev == next
}

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
