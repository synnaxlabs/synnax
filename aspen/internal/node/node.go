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
