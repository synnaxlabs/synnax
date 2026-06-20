// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Handle } from "@/arc/ir/types.gen";

// EDGE_KEY_SEP separates the four handle components of an edge key. Node keys
// (generated ids) and parameter names (identifiers) never contain a colon, so
// the key parses back into its endpoints unambiguously.
const EDGE_KEY_SEP = "::";

// edgeKey derives a stable identifier for an edge from its endpoints. Arc edges
// carry no key of their own, so the source and target handles serve as identity.
export const edgeKey = (source: Handle, target: Handle): string =>
  [source.node, source.param, target.node, target.param].join(EDGE_KEY_SEP);

// parseEdgeKey recovers the source and target handles from an edge key produced
// by edgeKey.
export const parseEdgeKey = (key: string): { source: Handle; target: Handle } => {
  const [sn, sp, tn, tp] = key.split(EDGE_KEY_SEP);
  return { source: { node: sn, param: sp }, target: { node: tn, param: tp } };
};
