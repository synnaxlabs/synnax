// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"time"

	"github.com/synnaxlabs/x/errors"
)

type Request struct {
	Message string `json:"message" msgpack:"message"`
	ID      int    `json:"id"      msgpack:"id"`
}

type Response struct {
	Message string `json:"message" msgpack:"message"`
	ID      int    `json:"id"      msgpack:"id"`
}

const WriteDeadline = 20 * time.Millisecond

var ErrCustom = errors.New("my custom error")

// ErrorProvider encodes and decodes ErrCustom for an errors.Codec.
func ErrorProvider() errors.Provider {
	return errors.Provider{
		Encode: func(_ *errors.Codec, err error) (errors.Payload, bool) {
			if errors.Is(err, ErrCustom) {
				return errors.Payload{Type: "myCustomError", Data: err.Error()}, true
			}
			return errors.Payload{}, false
		},
		Decode: func(_ *errors.Codec, f errors.Payload) (error, bool) {
			if f.Type != "myCustomError" {
				return nil, false
			}
			return ErrCustom, true
		},
	}
}
