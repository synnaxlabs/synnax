// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bus

import (
	"context"
	"encoding/hex"
	"uuid"

	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// MaxFrameSize is the size in bytes of the largest frame the Driver can hold.
const MaxFrameSize = 64 * 1024

// CheckFraming returns a validation error with a path to each field of a sync framing
// that describes frames the Driver cannot split. Rules on a single field live in the
// schema.
func CheckFraming(f Framing) error {
	s, ok := f.Variant.(SyncFraming)
	if !ok {
		return nil
	}
	v := validate.New("bus.framing")
	sync, err := hex.DecodeString(s.Sync)
	v.Ternary("sync", err != nil, "must be pairs of hex digits, such as AA55")
	v.Ternary(
		"length_size",
		s.LengthSize != 1 && s.LengthSize != 2 && s.LengthSize != 4,
		"must be 1, 2, or 4 bytes",
	)
	v.Ternary(
		"length_offset",
		int(s.LengthOffset) < len(sync),
		"must not overlap the sync sequence",
	)
	minSize := uint64(s.LengthOffset) + uint64(s.LengthSize) + checksumSize(s.Checksum)
	if minSize > MaxFrameSize {
		v.Ternaryf(
			"length_offset",
			true,
			"must leave room for the length field within %d bytes",
			MaxFrameSize,
		)
		return v.Error()
	}
	v.Ternaryf(
		"max_length",
		s.MaxLength != 0 && uint64(s.MaxLength) < minSize,
		"must be 0 or at least %d bytes",
		minSize,
	)
	return v.Error()
}

// StreamRead resolves cfg as Read does, then checks its framing with CheckFraming.
func (r Resolver) StreamRead(
	ctx context.Context,
	tx gorp.Tx,
	taskKey uuid.UUID,
	cfg *StreamReadConfig,
) error {
	if err := r.Read(ctx, tx, taskKey, &cfg.ReadConfig); err != nil {
		return err
	}
	return validate.PathedError(CheckFraming(cfg.Framing), "framing")
}

// StreamWrite resolves cfg as Write does, then checks its framing with CheckFraming.
func (r Resolver) StreamWrite(
	ctx context.Context,
	tx gorp.Tx,
	taskKey uuid.UUID,
	cfg *StreamWriteConfig,
) error {
	if err := r.Write(ctx, tx, taskKey, &cfg.WriteConfig); err != nil {
		return err
	}
	return validate.PathedError(CheckFraming(cfg.Framing), "framing")
}

func checksumSize(c Checksum) uint64 {
	switch c {
	case ChecksumCrc16Ccitt, ChecksumCrc16Modbus:
		return 2
	case ChecksumCrc32:
		return 4
	default:
		return 0
	}
}
