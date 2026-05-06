// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import xhttp "github.com/synnaxlabs/x/http"

// Encoder is an alias for x/http.Encoder, re-exported here so callers configuring
// freighter HTTP transports do not need to import x/http directly.
type Encoder = xhttp.Encoder

// Decoder is an alias for x/http.Decoder, re-exported here so callers configuring
// freighter HTTP transports do not need to import x/http directly.
type Decoder = xhttp.Decoder

// Codec is an alias for x/http.Codec, re-exported here so callers configuring
// freighter HTTP transports do not need to import x/http directly.
type Codec = xhttp.Codec
