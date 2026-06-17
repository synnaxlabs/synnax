// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { xy } from "@synnaxlabs/x";

// ZERO_GRAPH is an empty Arc graph. The editor owns graph state via the flux
// store, so this is only used as a fallback when a graph has not yet loaded.
export const ZERO_GRAPH: arc.graph.Graph = {
  nodes: [],
  edges: [],
  viewport: { position: xy.ZERO, zoom: 1 },
  functions: [],
};
