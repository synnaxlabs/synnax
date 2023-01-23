// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter/ferrors"
	"io"
)

const freighterError ferrors.Type = "freighter"

var (
	// EOF is returned when either the receiving or sending end of a Stream
	// exits normally.
	EOF = ferrors.Typed(io.EOF, freighterError)
	// StreamClosed is returned when a caller attempts to send or receive a message
	// from a stream that is already closed.
	StreamClosed = ferrors.Typed(errors.New("[freighter] - stream closed"), freighterError)
)

func encodeErr(err error) string { return err.Error() }

func decodeErr(encoded string) error {
	switch encoded {
	case EOF.Error():
		return EOF
	case StreamClosed.Error():
		return StreamClosed
	}
	panic("unknown error")
}

func init() {
	ferrors.Register(freighterError, encodeErr, decodeErr)
}
