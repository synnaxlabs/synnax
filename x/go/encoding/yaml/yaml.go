// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package yaml provides a YAML implementation of encoding.Codec and http.Codec for use
// as a portable text format (import/export bodies). It is a thin wrapper over gopkg.in/
// yaml.v3, so types customize their wire shape with the standard yaml.Marshaler /
// yaml.Unmarshaler interfaces and `yaml` struct tags. Note that yaml.v3 ignores `json`
// tags and lowercases untagged field names, so types serialized through this codec must
// carry explicit `yaml` tags (or implement yaml.Marshaler).
package yaml

import (
	"context"
	"io"

	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/http"
	"gopkg.in/yaml.v3"
)

// Codec is a YAML implementation of encoding.Codec and http.Codec.
var Codec http.Codec = &codec{}

type codec struct{}

func (c *codec) ContentType() string { return "application/yaml" }

func (c *codec) Encode(_ context.Context, value any) ([]byte, error) {
	b, err := yaml.Marshal(value)
	if err != nil {
		return nil, encoding.SugarEncodingErr(value, err)
	}
	return b, nil
}

func (c *codec) Decode(_ context.Context, data []byte, value any) error {
	if err := yaml.Unmarshal(data, value); err != nil {
		return encoding.SugarDecodingErr(data, value, err)
	}
	return nil
}

func (c *codec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := yaml.NewDecoder(r).Decode(value); err != nil {
		return encoding.SugarDecodingErr(nil, value, err)
	}
	return nil
}

func (c *codec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	if err := yaml.NewEncoder(w).Encode(value); err != nil {
		return encoding.SugarEncodingErr(value, err)
	}
	return nil
}
