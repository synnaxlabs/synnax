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
	"io"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

// ErrStreamClosed is returned when a caller attempts to send or receive a message
// from a stream that is already closed.
var ErrStreamClosed = errors.New("[freighter] - stream closed")

// ErrInterrupted is returned when the server stops while it handles a request. The
// request may or may not have taken effect, so it is not safe to resend it.
var ErrInterrupted = errors.New("[freighter] - server stopped during the request")

const (
	freighterErrorType    = "freighter."
	eofErrorType          = freighterErrorType + "eof"
	streamClosedErrorType = freighterErrorType + "stream_closed"
	interruptedErrorType  = freighterErrorType + "interrupted"
)

func encodeErr(_ *errors.Codec, err error) (errors.Payload, bool) {
	if errors.CheapIs(err, io.EOF) {
		return errors.Payload{Type: eofErrorType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrStreamClosed) {
		return errors.Payload{Type: streamClosedErrorType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrInterrupted) {
		return errors.Payload{Type: interruptedErrorType, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decodeErr(_ *errors.Codec, pld errors.Payload) (error, bool) {
	switch pld.Type {
	case eofErrorType:
		return io.EOF, true
	case streamClosedErrorType:
		return ErrStreamClosed, true
	case interruptedErrorType:
		return ErrInterrupted, true
	}
	if strings.HasPrefix(pld.Type, freighterErrorType) {
		return errors.New(pld.Data), true
	}
	return nil, false
}

// ErrorProvider encodes and decodes freighter errors for an errors.Codec.
func ErrorProvider() errors.Provider {
	return errors.Provider{Encode: encodeErr, Decode: decodeErr}
}
