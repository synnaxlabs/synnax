// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex

import (
	"bytes"
	"context"
	"encoding/json"

	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
)

// structToMap reduces a struct value to a flat map[string]any with exactly the
// semantics of encoding/json.Marshal — driven by `json:"..."` struct tags, with the
// Go field name as the fallback wire key. Anonymous (embedded) struct fields are
// inlined per the standard rules; `json:"-"` fields are skipped; `,omitempty`
// suppresses zero-valued entries.
//
// Implementation: marshal v with encoding/json, then decode the bytes back into a
// map[string]any via xjson.Codec so that downstream numeric values keep int64
// precision (UseNumber). The map is the codec-independent intermediate Encode merges
// headers into; it is never the wire output itself.
func structToMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "structToMap: marshal struct")
	}
	if bytes.Equal(b, []byte("null")) {
		return nil, errors.New("structToMap: value marshals to JSON null")
	}
	var m map[string]any
	if err := xjson.Codec.Decode(context.Background(), b, &m); err != nil {
		return nil, errors.Wrap(err, "structToMap: decode into map")
	}
	if m == nil {
		return nil, errors.New("structToMap: value does not marshal to a JSON object")
	}
	return m, nil
}
