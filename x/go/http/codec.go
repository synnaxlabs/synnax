// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Codec is an interface that extends encoding.Codec to
// add an HTTP content-type.
type Codec interface {
	ContentType() string
	encoding.Codec
}

var codecs = []Codec{json.Codec, msgpack.Codec}

type CodecResolver func(contentType string) (Codec, error)

func ResolveCodec(contentType string) (Codec, error) {
	for _, codec := range codecs {
		if codec.ContentType() == contentType {
			return codec, nil
		}
	}
	return nil, errors.Newf("[encoding] - unable to determine encoding type for %s", contentType)
}

var _ CodecResolver = ResolveCodec

func SupportedContentTypes() []string {
	var contentTypes []string
	for _, codec := range codecs {
		contentTypes = append(contentTypes, codec.ContentType())
	}
	return contentTypes
}
