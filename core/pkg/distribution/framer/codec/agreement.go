// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec

import (
	"context"
	"math"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/bit"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const (
	typesDeclaredFlagPos    bit.FlagPos = 0
	overridesPresentFlagPos bit.FlagPos = 1
	clocksDeclaredFlagPos   bit.FlagPos = 3
	// reservedExtensionFlags are the extension bits that no codec sets yet.
	reservedExtensionFlags byte        = 0b1111_0100
	typeKindPos            bit.FlagPos = 0
	clockKindPos           bit.FlagPos = 1
	// reservedKinds are the override kinds that no codec sets yet.
	reservedKinds byte = 0b1111_1100
	// noOverride is the next override ordinal that ends the chain.
	noOverride uint32 = math.MaxUint32
)

// state is the agreement that one sequence number names.
type state struct {
	// keys holds the requested keys, ascending.
	keys channel.Keys
	// index maps each requested key to its position in keys.
	index map[channel.Key]int
	// enc and dec hold what the frames of the state agreed on, as the encoding and the
	// decoding side see them.
	enc, dec agreement
}

// agreement holds the values that the frames of one state agree on.
type agreement struct {
	// typesDeclared is true once a frame of the state declared the agreed types.
	typesDeclared bool
	// dropped is true once a frame of the state failed to decode.
	dropped bool
	// variable is true when the agreed type of a requested key is variable.
	variable bool
	// dataTypes holds the agreed type of each requested key, in key order.
	dataTypes []telem.DataType
	// clocks holds the agreed clock of each requested key, in key order. Zero is no
	// clock.
	clocks []uint32
	// declared holds the declared clock keys, ascending. Each is a timestamp key that
	// is its own clock.
	declared channel.Keys
	// all is the ascending union of the requested and the declared keys.
	all channel.Keys
}

// newState returns the state of the given keys and their data types, in any order. A
// key named twice counts once. The data types are the encoder's agreed types; the
// decoder reads its own from the first frame of the state.
func newState(keys channel.Keys, dataTypes []telem.DataType) *state {
	sorted := slices.Compact(slices.Sorted(slices.Values(keys)))
	n := len(sorted)
	s := &state{
		keys:  sorted,
		index: make(map[channel.Key]int, n),
		enc: agreement{
			dataTypes: make([]telem.DataType, n),
			clocks:    make([]uint32, n),
			all:       sorted,
		},
		dec: agreement{
			dataTypes: make([]telem.DataType, n),
			clocks:    make([]uint32, n),
			all:       sorted,
		},
	}
	for i, key := range sorted {
		s.index[key] = i
	}
	for i, key := range keys {
		s.enc.dataTypes[s.index[key]] = dataTypes[i]
	}
	s.enc.variable = slices.ContainsFunc(s.enc.dataTypes, telem.DataType.IsVariable)
	return s
}

func isInt64Like(dt telem.DataType) bool {
	return dt == telem.Int64T || dt == telem.TimestampT
}

// isOwnClock returns true when s is a time series under its own key, which a frame
// carries as a declared clock when the request does not name the key.
func isOwnClock(key channel.Key, s telem.Series) bool {
	return s.Clock == uint32(key) && isInt64Like(s.DataType)
}

// union returns the ascending union of a and b, which are ascending and disjoint.
func union(a, b channel.Keys) channel.Keys {
	if len(b) == 0 {
		return a
	}
	out := make(channel.Keys, 0, len(a)+len(b))
	for len(a) > 0 && len(b) > 0 {
		if a[0] < b[0] {
			out, a = append(out, a[0]), a[1:]
		} else {
			out, b = append(out, b[0]), b[1:]
		}
	}
	return append(append(out, a...), b...)
}

// needsTypeOverride returns true when a series of type dt on a key of agreed type
// agreed must carry its own type code.
func needsTypeOverride(agreed, dt telem.DataType) bool {
	return agreed != dt && (!isInt64Like(agreed) || !isInt64Like(dt))
}

// codeOf returns the type code of dt, or an error naming key when dt has none.
func (c *Codec) codeOf(
	ctx context.Context,
	key channel.Key,
	dt telem.DataType,
) (byte, error) {
	code := telem.CodeOf(dt)
	if code == telem.DataTypeCodeUnknown {
		return 0, errors.Wrapf(
			validate.ErrValidation,
			"data type %q of channel %s has no type code",
			dt,
			c.retrieveName(ctx, key),
		)
	}
	return byte(code), nil
}

// frameDiff holds what the series of a frame change in the agreement of its state.
type frameDiff struct {
	// clocks is not zero when a series has a clock other than its key's agreed clock.
	clocks uint32
	// types is true when a series needs a type override.
	types bool
	// declared is true when the frame holds a declared clock.
	declared bool
}

// steady returns true when the frame needs no override.
func (d frameDiff) steady() bool { return d.clocks == 0 && !d.types }

// plan returns the extension flags of the next frame of ag, and fills c.codes, c.kinds,
// and c.clocks with its type codes, its override kinds, and the agreed clocks it
// leaves. It does not change ag.
func (c *Codec) plan(
	ctx context.Context,
	merged []mergedSeriesInfo,
	keys channel.Keys,
	ag *agreement,
	diff frameDiff,
) (byte, error) {
	var ext byte
	c.codes = c.codes[:0]
	if !ag.typesDeclared {
		ext = typesDeclaredFlagPos.Set(ext, true)
		for i, dt := range ag.dataTypes {
			code, err := c.codeOf(ctx, keys[i], dt)
			if err != nil {
				return 0, err
			}
			c.codes = append(c.codes, code)
		}
	}
	c.kinds = c.kinds[:0]
	clocks := ag.clocks
	if !diff.steady() {
		var err error
		if ext, err = c.planOverrides(ctx, merged, ag, ext); err != nil {
			return 0, err
		}
		clocks = c.clocks
	}
	if diff.declared || len(ag.declared) > 0 {
		c.planDeclared(merged, clocks, ag)
		if !slices.Equal(c.declared, ag.declared) {
			ext = clocksDeclaredFlagPos.Set(ext, true)
		}
	}
	return ext, nil
}

// planOverrides sets the override kinds of each series in c.kinds and the agreed clocks
// that the frame leaves in c.clocks.
func (c *Codec) planOverrides(
	ctx context.Context,
	merged []mergedSeriesInfo,
	ag *agreement,
	ext byte,
) (byte, error) {
	ext = overridesPresentFlagPos.Set(ext, true)
	c.clocks = append(c.clocks[:0], ag.clocks...)
	for _, msi := range merged {
		var kinds byte
		s := msi.series
		if msi.index < 0 {
			c.kinds = append(c.kinds, kinds)
			continue
		}
		if needsTypeOverride(ag.dataTypes[msi.index], s.DataType) {
			if _, err := c.codeOf(ctx, msi.key, s.DataType); err != nil {
				return 0, err
			}
			kinds = typeKindPos.Set(kinds, true)
		}
		if s.Clock != c.clocks[msi.index] {
			kinds = clockKindPos.Set(kinds, true)
			c.clocks[msi.index] = s.Clock
		}
		c.kinds = append(c.kinds, kinds)
	}
	return ext, nil
}

// planDeclared fills c.declared with the declared clocks of the frame: the key of each
// declared series in it, and each key of ag.declared that clocks still names.
func (c *Codec) planDeclared(
	merged []mergedSeriesInfo,
	clocks []uint32,
	ag *agreement,
) {
	c.declared = c.declared[:0]
	kept := 0
	for _, msi := range merged {
		if msi.index >= 0 ||
			(len(c.declared) > 0 && c.declared[len(c.declared)-1] == msi.key) {
			continue
		}
		for ; kept < len(ag.declared) && ag.declared[kept] < msi.key; kept++ {
			if slices.Contains(clocks, uint32(ag.declared[kept])) {
				c.declared = append(c.declared, ag.declared[kept])
			}
		}
		if kept < len(ag.declared) && ag.declared[kept] == msi.key {
			kept++
		}
		c.declared = append(c.declared, msi.key)
	}
	for ; kept < len(ag.declared); kept++ {
		if slices.Contains(clocks, uint32(ag.declared[kept])) {
			c.declared = append(c.declared, ag.declared[kept])
		}
	}
}

// planSize returns the bytes that the extension flags, the type codes, and the
// overrides of the planned frame take.
func (c *Codec) planSize(ext byte) int {
	if ext == 0 {
		return 0
	}
	size := 1 + len(c.codes)
	if clocksDeclaredFlagPos.Get(ext) {
		size += 4 + 4*len(c.declared)
	}
	if overridesPresentFlagPos.Get(ext) {
		size += 4
	}
	for _, kinds := range c.kinds {
		if kinds == 0 {
			continue
		}
		size += 1 + 4
		if typeKindPos.Get(kinds) {
			size++
		}
		if clockKindPos.Get(kinds) {
			size += 4
		}
	}
	return size
}

// nextOverride returns the ordinal of the first series from ordinal on that has an
// override, or noOverride.
func (c *Codec) nextOverride(ordinal int) uint32 {
	for ; ordinal < len(c.kinds); ordinal++ {
		if c.kinds[ordinal] != 0 {
			return uint32(ordinal)
		}
	}
	return noOverride
}

// writeOverride writes the override of the series at ordinal, if it has one.
func (c *Codec) writeOverride(ordinal int, s telem.Series) {
	if c.kinds[ordinal] == 0 {
		return
	}
	kinds := c.kinds[ordinal]
	c.buf.Uint8(kinds)
	if typeKindPos.Get(kinds) {
		c.buf.Uint8(byte(telem.CodeOf(s.DataType)))
	}
	if clockKindPos.Get(kinds) {
		c.buf.Uint32(s.Clock)
	}
	c.buf.Uint32(c.nextOverride(ordinal + 1))
}

// commit makes the planned frame the agreement of ag. The encoder calls it only after
// the whole frame encodes.
func (c *Codec) commit(ext byte, ag *agreement) {
	ag.typesDeclared = true
	if overridesPresentFlagPos.Get(ext) {
		copy(ag.clocks, c.clocks)
	}
	if clocksDeclaredFlagPos.Get(ext) {
		ag.declared = slices.Clone(c.declared)
		ag.all = c.all
	}
}

// frameKeys returns the key list of the planned frame of ag. It allocates only when the
// frame changes the declared clocks.
func (c *Codec) frameKeys(ext byte, keys channel.Keys, ag *agreement) channel.Keys {
	if clocksDeclaredFlagPos.Get(ext) {
		c.all = union(keys, c.declared)
		return c.all
	}
	return ag.all
}

func invalidf(format string, args ...any) error {
	return errors.Wrapf(validate.ErrValidation, format, args...)
}

// readDataType reads one type code and returns its data type.
func (c *Codec) readDataType() (telem.DataType, error) {
	code, err := c.reader.Uint8()
	if err != nil {
		return "", err
	}
	dt := telem.DataTypeCode(code).DataType()
	if dt == telem.UnknownT {
		return "", invalidf(
			"remote sent data type code %d, which this codec does not know",
			code,
		)
	}
	return dt, nil
}

// readExtension reads the extension flags, the type declaration, and the declared
// clocks of an extended frame.
func (c *Codec) readExtension(h *decodeHeader) error {
	var err error
	if h.ext, err = c.reader.Uint8(); err != nil {
		return err
	}
	if h.ext == 0 {
		return invalidf("remote sent extension flags of 0")
	}
	if h.ext&reservedExtensionFlags != 0 {
		return invalidf(
			"remote sent extension flags %08b, which hold a reserved bit",
			h.ext,
		)
	}
	if typesDeclaredFlagPos.Get(h.ext) {
		err = c.readTypes(h)
	} else {
		err = checkTypesDeclared(h)
	}
	if err != nil {
		return err
	}
	if clocksDeclaredFlagPos.Get(h.ext) {
		return c.readDeclared(h)
	}
	return nil
}

// checkTypesDeclared fails when no earlier frame of the state declared its types.
func checkTypesDeclared(h *decodeHeader) error {
	if !h.state.dec.typesDeclared {
		return noTypesError(h.seqNum)
	}
	return nil
}

func noTypesError(seqNum uint32) error {
	return invalidf(
		"remote sent no type declaration for state %d, so it runs an older codec, "+
			"which this codec does not support",
		seqNum,
	)
}

// readTypes reads the type declaration of the frame.
func (c *Codec) readTypes(h *decodeHeader) error {
	ag := &h.state.dec
	if ag.typesDeclared {
		return invalidf("remote declared the types of state %d twice", h.seqNum)
	}
	for i := range ag.dataTypes {
		dt, err := c.readDataType()
		if err != nil {
			return err
		}
		ag.dataTypes[i] = dt
	}
	ag.typesDeclared = true
	return nil
}

// readDeclared reads the declared clocks of the frame, which become the list of the
// state. It counts SeriesOverhead for each against the decode limit before it
// allocates, and it grows its buffer only as bytes arrive.
func (c *Codec) readDeclared(h *decodeHeader) error {
	n, err := c.reader.Uint32()
	if err != nil {
		return err
	}
	h.decoded += int64(n) * SeriesOverhead
	if err = c.checkDecodeLimit(h); err != nil {
		return err
	}
	b, err := c.readBytes(4 * int64(n))
	if err != nil {
		return err
	}
	declared := make(channel.Keys, n)
	for i := range declared {
		key := channel.Key(byteOrder.Uint32(b[4*i:]))
		if _, requested := h.state.index[key]; requested {
			return invalidf("remote declared clock %d, which the request names", key)
		}
		if i > 0 && key <= declared[i-1] {
			return invalidf(
				"remote declared clock %d after clock %d, so the list is not ascending",
				key,
				declared[i-1],
			)
		}
		declared[i] = key
	}
	ag := &h.state.dec
	ag.declared = declared
	ag.all = union(h.state.keys, declared)
	return nil
}

// checkDecodeLimit fails when the frame decodes to more than the decode limit.
func (c *Codec) checkDecodeLimit(h *decodeHeader) error {
	if c.decodeLimit > 0 && h.decoded > c.decodeLimit {
		return limitError(c.decodeLimit)
	}
	return nil
}

func limitError(limit int64) error {
	return invalidf("frame decodes to more than the limit of %d bytes", limit)
}

// readSeriesAgreement returns the data type and the clock of the series of key at
// h.ordinal, and reads its override when it has one.
func (c *Codec) readSeriesAgreement(
	h *decodeHeader,
	key channel.Key,
) (telem.DataType, uint32, error) {
	ag := &h.state.dec
	if i, ok := h.state.index[key]; ok {
		dt := ag.dataTypes[i]
		if h.ordinal == h.next {
			var err error
			if dt, err = c.readOverride(h, i); err != nil {
				return "", 0, err
			}
		}
		return dt, ag.clocks[i], nil
	}
	if _, ok := slices.BinarySearch(ag.declared, key); !ok {
		return "", 0, errors.Newf("unknown channel key: %v", key)
	}
	if h.ordinal == h.next {
		if _, err := c.readKinds(h); err != nil {
			return "", 0, err
		}
		return "", 0, invalidf("remote sent an override for declared clock %d", key)
	}
	return telem.TimestampT, uint32(key), nil
}

// readKinds reads the override kinds of the series at h.ordinal.
func (c *Codec) readKinds(h *decodeHeader) (byte, error) {
	kinds, err := c.reader.Uint8()
	if err != nil {
		return 0, err
	}
	if kinds == 0 {
		return 0, invalidf("remote sent override kinds of 0 for series %d", h.ordinal)
	}
	if kinds&reservedKinds != 0 {
		return 0, invalidf(
			"remote sent override kinds %08b for series %d, which hold a reserved bit",
			kinds,
			h.ordinal,
		)
	}
	return kinds, nil
}

// readOverride reads the override of the series at h.ordinal, of the requested key at
// index i, and returns the series' data type.
func (c *Codec) readOverride(h *decodeHeader, i int) (telem.DataType, error) {
	ag := &h.state.dec
	dt := ag.dataTypes[i]
	kinds, err := c.readKinds(h)
	if err != nil {
		return "", err
	}
	if typeKindPos.Get(kinds) {
		if dt, err = c.readDataType(); err != nil {
			return "", err
		}
	}
	if clockKindPos.Get(kinds) {
		if ag.clocks[i], err = c.reader.Uint32(); err != nil {
			return "", err
		}
	}
	if h.next, err = c.reader.Uint32(); err != nil {
		return "", err
	}
	if h.next != noOverride && h.next <= h.ordinal {
		return "", invalidf(
			"remote sent next override %d, which is not after series %d",
			h.next,
			h.ordinal,
		)
	}
	return dt, nil
}
