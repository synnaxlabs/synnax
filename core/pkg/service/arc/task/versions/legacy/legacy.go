// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy marks the Arc task config shape released Consoles stored and
// exported: {arcKey}. It needs only era normalization.
package legacy

import "github.com/synnaxlabs/synnax/pkg/service/imex"

// LastVersion is the newest legacy Arc task shape. The typed shape sits directly
// above it.
const LastVersion imex.Version = 0
