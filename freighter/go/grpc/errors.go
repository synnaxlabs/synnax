// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import "github.com/synnaxlabs/x/errors"

// EncodeError encodes err with codec into a protobuf error payload. See
// errors.Codec.Encode for the payload of an error that codec does not handle.
func EncodeError(codec *errors.Codec, err error, internal bool) *errors.PBPayload {
	pld := codec.Encode(err, internal)
	return &errors.PBPayload{Type: pld.Type, Data: pld.Data}
}
