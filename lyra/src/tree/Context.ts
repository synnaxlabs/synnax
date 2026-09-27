// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@/context";
import { type NodeShape } from "@/tree/base";

export interface ContextValue {
  nodes: NodeShape[];
  /** Set when carets toggle their nodes, by index; a row click then only selects. */
  toggle?: (index: number) => void;
}

export const [Context, useContext] = context.create<ContextValue>({
  displayName: "Tree.Context",
  providerName: "Tree.Provider",
});
