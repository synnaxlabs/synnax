// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy names the Console-era symbol wire formats. Each subpackage owns one
// frozen shape; this package owns the boundary between them and the server-exported
// versions. Symbol was never persisted as an opaque state blob, so there is no
// migration chain here — v1 is the single Console format, decoded directly by ImEx.
package legacy

import (
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/legacy/v1"
)

// LastVersion is the highest version a Console-written file carries. Envelopes stamped
// above it are server exports and route to the generated migration chain.
const LastVersion = v1.Version
