// Copyright 2026 Synnax Labs, Inc.
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

// Codec is an interface that extends binary.Codec to
// add an HTTP content-type.
type Codec interface {
	ContentType() string
	binary.Codec
}

type typedCodec struct {
	ct string
	binary.Codec
}

func (t typedCodec) ContentType() string { return t.ct }

var (
	JSONCodec = typedCodec{
		ct:    "application/json",
		Codec: &binary.JSONCodec{},
	}
	MsgPackCodec = typedCodec{
		ct:    "application/msgpack",
		Codec: &binary.MsgPackCodec{},
	}
)

var codecs = []Codec{JSONCodec, MsgPackCodec}

type CodecResolver func(contentType string) (Codec, error)

func ResolveCodec(contentType string) (Codec, error) {
	for _, ecd := range codecs {
		if ecd.ContentType() == contentType {
			return ecd, nil
		}
	}
	return nil, errors.Newf("[encoding] - unable to determine encoding type for %s", contentType)
}

var _ CodecResolver = ResolveCodec

func SupportedContentTypes() []string {
	var contentTypes []string
	for _, ecd := range codecs {
		contentTypes = append(contentTypes, ecd.ContentType())
	}
	return contentTypes
}
