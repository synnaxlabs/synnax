// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import (
	"io"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/encoding/leb128"
)

func writeUnsignedLeb128(w io.ByteWriter, val uint64) {
	lo.Must0(leb128.WriteUnsigned(w, val))
}

func writeSignedLeb128(w io.ByteWriter, val int64) {
	lo.Must0(leb128.WriteSigned(w, val))
}
