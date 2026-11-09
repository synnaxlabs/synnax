// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// Marshal encodes a document with its object names in sorted order. Invalid UTF-8
// becomes the replacement character.
func Marshal(doc any) ([]byte, error) {
	return json.Marshal(doc, json.Deterministic(true), jsontext.AllowInvalidUTF8(true))
}
