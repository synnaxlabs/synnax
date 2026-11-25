// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import type { Spec } from "@/schematic/node/spec";
import { Node, Preview } from "@/schematic/node/symbol/Symbol";

export const SYMBOL_VARIANT = "symbol";

/**
 * symbolSpec draws a symbol node: its symbol, with the looks that the values of its
 * bound states give it, and the stale indicator while a state is stale.
 */
export const symbolSpec: Spec<typeof SYMBOL_VARIANT, schematic.SymbolNodeConfig> = {
  key: SYMBOL_VARIANT,
  name: "Symbol",
  Form: () => null,
  Node,
  Preview,
  zIndex: 4,
};
