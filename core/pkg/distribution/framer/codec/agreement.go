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
	// dataTypes holds the agreed type of each requested key, in key order.
	dataTypes []telem.DataType
	// clocks holds the agreed clock of each requested key, in key order. Zero is no
	// clock.
	clocks []uint32
}

// newState returns the state of the given keys and their data types, in any order. The
// data types are the encoder's agreed types; the decoder reads its own from the first
// frame of the state.
func newState(keys channel.Keys, dataTypes []telem.DataType) *state {
	s := &state{
		keys:  slices.Clone(keys),
		index: make(map[channel.Key]int, len(keys)),
		enc: agreement{
			dataTypes: make([]telem.DataType, len(keys)),
			clocks:    make([]uint32, len(keys)),
		},
		dec: agreement{
			dataTypes: make([]telem.DataType, len(keys)),
			clocks:    make([]uint32, len(keys)),
		},
	}
	slices.Sort(s.keys)
	for i, key := range s.keys {
		s.index[key] = i
	}
	for i, key := range keys {
		s.enc.dataTypes[s.index[key]] = dataTypes[i]
	}
	return s
}

func isInt64Like(dt telem.DataType) bool {
	return dt == telem.Int64T || dt == telem.TimestampT
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

// plan returns the extension flags of the next frame of ag, and fills c.codes,
// c.kinds, and c.clocks with its type codes, its override kinds, and the agreed clocks
// it leaves. It does not change ag.
func (c *Codec) plan(
	ctx context.Context,
	merged []mergedSeriesInfo,
	keys channel.Keys,
	ag *agreement,
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
	var (
		clockDiff   uint32
		typesDiffer bool
	)
	for _, msi := range merged {
		clockDiff |= msi.series.Clock ^ ag.clocks[msi.index]
		typesDiffer = typesDiffer ||
			needsTypeOverride(ag.dataTypes[msi.index], msi.series.DataType)
	}
	c.kinds = c.kinds[:0]
	if clockDiff == 0 && !typesDiffer {
		return ext, nil
	}
	ext = overridesPresentFlagPos.Set(ext, true)
	c.clocks = append(c.clocks[:0], ag.clocks...)
	for _, msi := range merged {
		var kinds byte
		s := msi.series
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

// planSize returns the bytes that the extension flags, the type codes, and the
// overrides of the planned frame take.
func (c *Codec) planSize(ext byte) int {
	if ext == 0 {
		return 0
	}
	size := 1 + len(c.codes)
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
	if ordinal >= len(c.kinds) || c.kinds[ordinal] == 0 {
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

// readExtension reads the extension flags and the type declaration of the frame into h.
func (c *Codec) readExtension(h *decodeHeader) error {
	ag := &h.state.dec
	if h.fgs.extended {
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
	}
	if !typesDeclaredFlagPos.Get(h.ext) {
		if !ag.typesDeclared {
			return invalidf(
				"remote sent no type declaration for state %d, so it runs an older "+
					"codec, which this codec does not support",
				h.seqNum,
			)
		}
		return nil
	}
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

// readOverride reads the override of the series at h.ordinal, of the requested key at
// index i, and returns the series' data type.
func (c *Codec) readOverride(h *decodeHeader, i int) (telem.DataType, error) {
	ag := &h.state.dec
	dt := ag.dataTypes[i]
	kinds, err := c.reader.Uint8()
	if err != nil {
		return "", err
	}
	if kinds == 0 {
		return "", invalidf("remote sent override kinds of 0 for series %d", h.ordinal)
	}
	if kinds&reservedKinds != 0 {
		return "", invalidf(
			"remote sent override kinds %08b for series %d, which hold a reserved bit",
			kinds,
			h.ordinal,
		)
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
