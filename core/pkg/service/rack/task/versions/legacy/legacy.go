// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy marks the rack status config shape released Cores stored and
// exported. It needs only era normalization.
package legacy

import "github.com/synnaxlabs/synnax/pkg/service/imex"

// LastVersion is the newest version whose blobs need the rewrite. Released Cores
// stamped the v0 shape at 1, and era normalization leaves it unchanged.
const LastVersion imex.Version = 1
