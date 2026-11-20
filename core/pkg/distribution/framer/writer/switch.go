// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"

	"github.com/synnaxlabs/x/address"
)

// switchFree sends the stored channels of a request to the storage writer and the free
// channels to the free writer.
func switchFree(_ context.Context, r Request, oReqs map[address.Address]Request) error {
	stored, free := r.Frame.SplitFree()
	sr, fr := r, r
	sr.Frame, fr.Frame = stored, free
	oReqs[storageWriterAddr] = sr
	oReqs[freeWriterAddr] = fr
	return nil
}
