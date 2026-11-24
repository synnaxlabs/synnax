// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment

import (
	"fmt"
	"io"

	"github.com/synnaxlabs/x/errors"
)

// Search finds which of candidates is the config of the version 1 segment file in r of
// size bytes. It checks the full blocks in order until one passes for a candidate,
// which must be the only one to pass that block. It returns that config, and the number
// of full blocks from the start of the file that pass for it. When it finds no config,
// it returns why instead: the file holds no full block, no candidate passes, or two
// pass one block.
func Search(
	r io.ReaderAt,
	size int64,
	candidates []Config,
) (cfg Config, checked int64, reason string, err error) {
	buf := make([]byte, BlockSize+VariableTrailer)
	k := int64(0)
	for ; ; k++ {
		passing, full, err := passingAt(r, size, buf, k, candidates)
		switch {
		case err != nil:
			return Config{}, 0, "", err
		case !full && k == 0:
			return Config{}, 0, "it holds no full block", nil
		case !full:
			return Config{}, 0, "no candidate passes a full block", nil
		case len(passing) > 1:
			return Config{}, 0, fmt.Sprintf(
				"type %d with clock %d and type %d with clock %d both pass block %d",
				passing[0].DataType,
				passing[0].Clock,
				passing[1].DataType,
				passing[1].Clock,
				k,
			), nil
		}
		if len(passing) == 1 {
			cfg = passing[0]
			break
		}
	}
	if k > 0 {
		return cfg, 0, "", nil
	}
	for checked = 1; ; checked++ {
		passing, full, err := passingAt(r, size, buf, checked, []Config{cfg})
		if err != nil {
			return Config{}, 0, "", err
		}
		if !full || len(passing) == 0 {
			return cfg, checked, "", nil
		}
	}
}

// passingAt returns the candidates that full block k of the segment file in r of size
// bytes passes, and whether the file holds block k in full for a candidate. It reads
// into buf, which holds a block and the largest trailer.
func passingAt(
	r io.ReaderAt,
	size int64,
	buf []byte,
	k int64,
	candidates []Config,
) ([]Config, bool, error) {
	var (
		passing []Config
		full    bool
	)
	for _, variable := range []bool{false, true} {
		slot := BlockSize + Config{Variable: variable}.trailer()
		if (k+1)*slot > size {
			continue
		}
		read := false
		for _, c := range candidates {
			if c.Variable != variable {
				continue
			}
			full = true
			if !read {
				if _, err := r.ReadAt(buf[:slot], k*slot); err != nil {
					return nil, false, errors.Wrapf(err, "read of block %d", k)
				}
				read = true
			}
			if c.checkBlock(k, buf[:slot], BlockSize, 0) == nil {
				passing = append(passing, c)
			}
		}
	}
	return passing, full, nil
}
