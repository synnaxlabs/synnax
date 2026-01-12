// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

// TranslatePayloadForward translates the error payload to its protobuf representation.
func TranslatePayloadForward(pld Payload) *PBPayload {
	return &PBPayload{Type: pld.Type, Data: pld.Data}
}

// TranslatePayloadBackward translates the error payload to its standard go representation
// from its protobuf representation.
func TranslatePayloadBackward(pld *PBPayload) Payload {
	return Payload{Type: pld.Type, Data: pld.Data}
}
