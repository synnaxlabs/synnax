// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fgrpc

import (
	"context"
	"github.com/synnaxlabs/x/errors"
)

// EncodeError encodes the given error into a protobuf error payload. If the type of
// the error cannot be determined, returns a payload with type TypeUnknown and the error
// message. If the error is nil, returns a payload with type TypeNil.
func EncodeError(ctx context.Context, err error, internal bool) *errors.PBPayload {
	pld := errors.Encode(ctx, err, internal)
	return &errors.PBPayload{Type: pld.Type, Data: pld.Data}
}

// DecodeError decodes the given protobuf error payload into an error. If the payload's
// type is TypeUnknown, returns an error with the payload's data as the message. If the
// payload's type is TypeNil, returns nil.
func DecodeError(ctx context.Context, pld *errors.PBPayload) error {
	return errors.Decode(ctx, errors.Payload{Type: pld.Type, Data: pld.Data})
}
