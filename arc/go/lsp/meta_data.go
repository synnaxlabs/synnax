// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp

import (
	"strings"

	"go.lsp.dev/protocol"
)

type DocumentMetadata struct {
	IsFunctionBlock bool `json:"is_function_block"`
}

func ExtractMetadataFromURI(uri protocol.DocumentURI) *DocumentMetadata {
	uriStr := string(uri)
	return &DocumentMetadata{
		IsFunctionBlock: strings.HasPrefix(uriStr, "arc://block/"),
	}
}
