// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package marshal

import (
	"bytes"
	"text/template"

	"github.com/synnaxlabs/x/errors"
)

// FlexCodec describes a distinct scalar type (e.g., Key uint64) for which
// flexible DecodeMsgpack and UnmarshalJSON methods should be generated. These
// methods handle type coercion (float64 -> uint64, string -> uint64, etc.) so
// the type decodes correctly when used inside generic containers that fall back
// to msgpack/JSON reflection-based decoding.
type FlexCodec struct {
	GoName   string
	Receiver string
	BaseType string
}

type flexConfig struct {
	MsgpackHelper string
	JSONHelper    string
	CastType      string
}

var flexConfigMap = map[string]flexConfig{
	"uint64": {
		MsgpackHelper: "UnmarshalUint64",
		JSONHelper:    "UnmarshalStringUint64",
		CastType:      "uint64",
	},
	"uint32": {
		MsgpackHelper: "UnmarshalUint32",
		JSONHelper:    "UnmarshalStringUint32",
		CastType:      "uint32",
	},
}

type flexTemplateData struct {
	GoName        string
	Receiver      string
	MsgpackHelper string
	JSONHelper    string
	CastType      string
}

func generateFlexMethods(fc FlexCodec) (string, error) {
	cfg, ok := flexConfigMap[fc.BaseType]
	if !ok {
		return "", errors.Newf("unsupported flex base type %q for %s", fc.BaseType, fc.GoName)
	}
	data := flexTemplateData{
		GoName:        fc.GoName,
		Receiver:      fc.Receiver,
		MsgpackHelper: cfg.MsgpackHelper,
		JSONHelper:    cfg.JSONHelper,
		CastType:      cfg.CastType,
	}
	tmpl, err := template.New("flex").Parse(flexTemplate)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse flex template")
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", errors.Wrap(err, "failed to execute flex template")
	}
	return buf.String(), nil
}

const flexTemplate = `
func ({{.Receiver}} *{{.GoName}}) DecodeMsgpack(dec *msgpack.Decoder) error {
	n, err := xmsgpack.{{.MsgpackHelper}}(dec)
	if err != nil {
		return err
	}
	*{{.Receiver}} = {{.GoName}}(n)
	return nil
}

func ({{.Receiver}} *{{.GoName}}) UnmarshalJSON(b []byte) error {
	n, err := xjson.{{.JSONHelper}}(b)
	if err != nil {
		return err
	}
	*{{.Receiver}} = {{.GoName}}(n)
	return nil
}
`
