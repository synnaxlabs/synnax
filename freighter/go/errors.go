// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter

import (
	"context"
	"io"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

var (
	// EOF is returned when either the receiving or sending end of a Stream
	// exits normally.
	EOF = io.EOF
	// ErrStreamClosed is returned when a caller attempts to send or receive a message
	// from a stream that is already closed.
	ErrStreamClosed = errors.New("[freighter] - stream closed")
)

const (
	freighterErrorType    = "freighter."
	eofErrorType          = freighterErrorType + "eof"
	streamClosedErrorType = freighterErrorType + "stream_closed"
)

func encodeErr(_ context.Context, err error) (errors.Payload, bool) {
	if errors.Is(err, EOF) {
		return errors.Payload{Type: eofErrorType, Data: err.Error()}, true
	}
	if errors.Is(err, ErrStreamClosed) {
		return errors.Payload{Type: streamClosedErrorType, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decodeErr(_ context.Context, pld errors.Payload) (error, bool) {
	switch pld.Type {
	case eofErrorType:
		return EOF, true
	case streamClosedErrorType:
		return ErrStreamClosed, true
	}
	if strings.HasPrefix(pld.Type, freighterErrorType) {
		return errors.New(pld.Data), true
	}
	return nil, false
}

func init() {
	errors.Register(encodeErr, decodeErr)
}
