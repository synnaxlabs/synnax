// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"context"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// locationKey is a comparable form of a Location.
type locationKey struct {
	typ      LocationType
	address  uint16
	value    ValueType
	dataType telem.NumericDataType
	bit      uint8
	length   uint16
}

func keyOf(l Location) locationKey {
	var k locationKey
	var value Value
	switch v := l.Variant.(type) {
	case CoilLocation:
		k.typ, k.address = CoilLocationType, v.Address
	case DiscreteInputLocation:
		k.typ, k.address = DiscreteInputLocationType, v.Address
	case HoldingRegisterLocation:
		k.typ, k.address, value = HoldingRegisterLocationType, v.Address, v.Value
	case InputRegisterLocation:
		k.typ, k.address, value = InputRegisterLocationType, v.Address, v.Value
	}
	switch v := value.Variant.(type) {
	case WholeValue:
		k.value, k.dataType = WholeValueType, v.DataType
	case BitValue:
		k.value, k.bit = BitValueType, v.Bit
	case StringValue:
		k.value, k.length = StringValueType, v.Length
	}
	return k
}

// hostLabel matches one label of a hostname. It allows underscores, which Docker
// service names and some Windows machine names hold and which getaddrinfo resolves.
var hostLabel = regexp.MustCompile(`^[A-Za-z0-9_]([A-Za-z0-9_-]{0,61}[A-Za-z0-9_])?$`)

// isHost returns true if host is an IP address or a hostname whose last label is not
// all digits.
func isHost(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}
	labels := strings.Split(host, ".")
	if len(host) > 253 || strings.Trim(labels[len(labels)-1], "0123456789") == "" {
		return false
	}
	return lo.EveryBy(labels, hostLabel.MatchString)
}

// validateServer checks s against its schema. It also checks that a TCP host is a
// hostname or an IP address, that every value ends at or before address 65535, that no
// two reads share a location and an index, and that no two writes set one address.
func validateServer(s Server) error {
	if err := s.Validate(); err != nil {
		return err
	}
	v := validate.New("Server")
	if tcp, ok := s.Connection.Variant.(TcpConnection); ok {
		v.Ternary(
			"connection.host",
			!isHost(tcp.Host),
			"must be a hostname, an IPv4 address, or an IPv6 address",
		)
	}
	type readKey struct {
		location locationKey
		index    channel.Key
	}
	reads := set.New[readKey]()
	for i, r := range s.Reads {
		path := "reads." + strconv.Itoa(i)
		start, count := span(r.Location)
		v.Ternary(path+".location", !fits(start, count), overrunMessage)
		k := readKey{location: keyOf(r.Location), index: r.Index}
		v.Ternary(
			path,
			reads.Contains(k),
			"repeats the location and index of another read",
		)
		reads.Add(k)
	}
	var written writeSet
	for i, w := range s.Writes {
		path := "writes." + strconv.Itoa(i)
		start, count := writeSpan(w.Location)
		v.Ternary(path+".location", !fits(start, count), overrunMessage)
		v.Ternary(
			path,
			!written.add(w.Location),
			"sets an address that another write sets",
		)
	}
	return v.Error()
}

const overrunMessage = "must end at or before address 65535"

// fits returns true if count addresses from start all lie at or below 65535.
func fits(start uint16, count int) bool {
	return int(start)+count-1 <= math.MaxUint16
}

// span returns the first address and the number of addresses that a read of l covers.
func span(l Location) (uint16, int) {
	switch v := l.Variant.(type) {
	case CoilLocation:
		return v.Address, 1
	case DiscreteInputLocation:
		return v.Address, 1
	case HoldingRegisterLocation:
		return v.Address, valueCount(v.Value)
	case InputRegisterLocation:
		return v.Address, valueCount(v.Value)
	}
	return 0, 1
}

// valueCount returns the number of registers that v fills.
func valueCount(v Value) int {
	switch val := v.Variant.(type) {
	case WholeValue:
		return registerCount(val.DataType)
	case StringValue:
		return (int(val.Length) + 1) / 2
	}
	return 1
}

// writeSpan returns the first address and the number of addresses that a write of l
// sets.
func writeSpan(l WriteLocation) (uint16, int) {
	switch v := l.Variant.(type) {
	case CoilWriteLocation:
		return v.Address, 1
	case HoldingRegisterWriteLocation:
		if whole, ok := v.Value.Variant.(WholeWriteValue); ok {
			return v.Address, registerCount(whole.DataType)
		}
		return v.Address, 1
	}
	return 0, 1
}

// address is one address in one Modbus address space.
type address struct {
	typ     LocationType
	address int
}

// writeSet holds the coils, registers, and register bits that writes set.
type writeSet struct {
	whole set.Set[address]
	bits  map[address]set.Set[uint8]
}

// add records the addresses that a write of r sets. It returns false if another write
// already sets one of them: a coil, a register of a whole value, or the same bit or the
// whole of a register that a bit write sets.
func (ws *writeSet) add(l WriteLocation) bool {
	if ws.whole == nil {
		ws.whole, ws.bits = set.New[address](), map[address]set.Set[uint8]{}
	}
	switch v := l.Variant.(type) {
	case CoilWriteLocation:
		return ws.addWhole(address{typ: CoilLocationType, address: int(v.Address)}, 1)
	case HoldingRegisterWriteLocation:
		start := address{typ: HoldingRegisterLocationType, address: int(v.Address)}
		switch value := v.Value.Variant.(type) {
		case WholeWriteValue:
			return ws.addWhole(start, registerCount(value.DataType))
		case BitWriteValue:
			return ws.addBit(start, value.Bit)
		}
	}
	return true
}

func (ws *writeSet) addWhole(start address, count int) bool {
	free := true
	for i := range count {
		a := address{typ: start.typ, address: start.address + i}
		_, hasBits := ws.bits[a]
		free = free && !ws.whole.Contains(a) && !hasBits
		ws.whole.Add(a)
	}
	return free
}

func (ws *writeSet) addBit(a address, bit uint8) bool {
	bits, ok := ws.bits[a]
	if ws.whole.Contains(a) || (ok && bits.Contains(bit)) {
		return false
	}
	if !ok {
		bits = set.New[uint8]()
		ws.bits[a] = bits
	}
	bits.Add(bit)
	return true
}

// registerCount returns the number of 16-bit registers that a value of dt fills.
func registerCount(dt telem.NumericDataType) int {
	return max(1, int(telem.DataType(dt).Density())/2)
}

// Writer creates, updates, and deletes Modbus servers in the transaction given to
// Service.NewWriter.
type Writer struct {
	tx       gorp.Tx
	status   status.Writer
	statuses *status.Service
	table    *gorp.Table[Key, Server]
}

func (w Writer) setStatus(ctx context.Context, s Server) error {
	var st Status
	err := w.statuses.NewRetrieve[StatusDetails]().
		Where(status.MatchOwners[StatusDetails](s.ResourceID())).
		Entry(&st).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		st = Status{
			Owner:   new(s.ResourceID()),
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Status unknown",
			Details: StatusDetails{Server: s.Key},
		}
	} else if err != nil {
		return err
	}
	st.Name = s.Name
	st.Details.Rack = s.Rack
	return w.status.Set(ctx, &st)
}

// Create stores s and writes an unknown status when it has none. An existing status
// takes the name and rack of s. A server with a nil key gets a new one. It returns a
// validation error when s is invalid or when its rack does not exist.
func (w Writer) Create(ctx context.Context, s *Server) error {
	if s.Key == uuid.Nil() {
		s.Key = uuid.New()
	}
	if err := validateServer(*s); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.setStatus(ctx, *s)
}

// update applies change to the server stored under key. A rename or rack move updates
// the status, or writes it as unknown when it is missing.
func (w Writer) update(
	ctx context.Context,
	key Key,
	change func(Server) (Server, error),
) error {
	var prev, next Server
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Server](key)).
		ChangeErr(func(_ gorp.Context, s Server) (Server, error) {
			prev = s
			updated, err := change(s)
			if err != nil {
				return Server{}, err
			}
			updated.Key = key
			if err = validateServer(updated); err != nil {
				return Server{}, err
			}
			next = updated
			return updated, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if next.Name != prev.Name || next.Rack != prev.Rack {
		return w.setStatus(ctx, next)
	}
	return nil
}

// Delete removes the servers stored under keys with their statuses. Delete is
// idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Server](keys...)).
		Exec(ctx, w.tx)
}
