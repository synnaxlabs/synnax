// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package httputil

import (
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
)

// EncoderDecoder is an interface that extends binary.Codec to
// add an HTTP content-type.
type EncoderDecoder interface {
	ContentType() string
	binary.Codec
}

type typedEncoderDecoder struct {
	ct string
	binary.Codec
}

func (t typedEncoderDecoder) ContentType() string { return t.ct }

var (
	JSONEncoderDecoder = typedEncoderDecoder{
		ct:    "application/json",
		Codec: &binary.JSONEncoderDecoder{},
	}
	MsgPackEncoderDecoder = typedEncoderDecoder{
		ct:    "application/msgpack",
		Codec: &binary.MsgPackEncoderDecoder{},
	}
)

var encoderDecoders = []EncoderDecoder{
	JSONEncoderDecoder,
	MsgPackEncoderDecoder,
}

func DetermineEncoderDecoder(contentType string) (EncoderDecoder, error) {
	for _, ecd := range encoderDecoders {
		if ecd.ContentType() == contentType {
			return ecd, nil
		}
	}
	return nil, errors.New("[encoding] - unable to determine encoding type")
}

func SupportedContentTypes() []string {
	var contentTypes []string
	for _, ecd := range encoderDecoders {
		contentTypes = append(contentTypes, ecd.ContentType())
	}
	return contentTypes
}
