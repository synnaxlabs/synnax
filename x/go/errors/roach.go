// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

import (
	"context"
	"encoding/hex"

	"github.com/cockroachdb/errors"
)

func roachEncode(ctx context.Context, err error) Payload {
	// Case where the error is of type roach or not typed. If the type isn't registered,
	// attempt to encode the error using cockroachdb's error package. This is used for
	// go-to-go transport.
	encoded := errors.EncodeError(ctx, err)
	b, err := encoded.Marshal()
	// If we couldn't encode the error, return a standardized unknown payload along with
	// the error string.
	if err != nil {
		return Payload{Type: TypeUnknown, Data: err.Error()}
	}
	return Payload{Type: TypeRoach, Data: hex.EncodeToString(b)}
}

// Decode implements Provider.
func roachDecode(ctx context.Context, pld Payload) (error, bool) {
	if pld.Type != TypeRoach {
		return nil, false
	}
	e := &errors.EncodedError{}
	b, err := hex.DecodeString(pld.Data)
	if err != nil {
		return nil, false
	}
	if err := e.Unmarshal(b); err != nil {
		return nil, false
	}
	return errors.DecodeError(ctx, *e), true
}
