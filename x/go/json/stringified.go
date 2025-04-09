// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"context"
	"encoding/json"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

type String string

var detailsCodec = &binary.JSONCodec{}

func NewStaticString(ctx context.Context, data interface{}) String {
	b, err := detailsCodec.Encode(ctx, data)
	if err != nil {
		zap.S().DPanic("unexpected static encode error", zap.Error(err))
	}
	return String(b)
}

// UnmarshalJSON implements the json.Unmarshaler interface for String.
// It should correctly handle a raw JSON string or a JSON object/array.
func (d *String) UnmarshalJSON(data []byte) error {
	var plainString string
	if err := json.Unmarshal(data, &plainString); err == nil {
		*d = String(plainString)
		return nil
	}
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return errors.Wrap(err, "failed to unmarshal JSON")
	}
	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	*d = String(bytes)
	return nil
}
