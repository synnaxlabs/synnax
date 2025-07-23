// Copyright 2025 Synnax Labs, Inc.
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

// Codec is an interface that extends binary.Codec to add an HTTP content-type.
type Codec interface {
	ContentType() string
	binary.Codec
}

type typedCodec struct {
	contentType string
	binary.Codec
}

func (tc typedCodec) ContentType() string { return tc.contentType }

var (
	JSONCodec = typedCodec{
		contentType: "application/json",
		Codec:       &binary.JSONCodec{},
	}
	MsgPackCodec = typedCodec{
		contentType: "application/msgpack",
		Codec:       &binary.MsgPackCodec{},
	}
)

var codecs = []Codec{JSONCodec, MsgPackCodec}

type CodecResolver func(string) (Codec, error)

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
	contentTypes := make([]string, len(codecs))
	for i, ecd := range codecs {
		contentTypes[i] = ecd.ContentType()
	}
	return contentTypes
}
