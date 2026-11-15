// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import {
  EDGE_CONFIG_SCHEMAS,
  type EdgeConfig,
  edgeConfigZ,
  NODE_CONFIG_SCHEMAS,
  type NodeConfig,
  nodeConfigZ,
} from "@/schematic/types.gen";

/** The config of a node or an edge. No node variant shares a name with an edge one. */
export const elementConfigZ = z.discriminatedUnion("variant", [
  ...nodeConfigZ.options,
  ...edgeConfigZ.options,
]);
export type ElementConfig = NodeConfig | EdgeConfig;
export type ElementConfigType = ElementConfig["variant"];

/** The schema of every node and edge config variant, by variant. */
export const ELEMENT_CONFIG_SCHEMAS = {
  ...NODE_CONFIG_SCHEMAS,
  ...EDGE_CONFIG_SCHEMAS,
};

/** The list of a schematic that holds a node or an edge. */
export type ElementField = "nodes" | "edges";

/** @returns the list that holds the element the config belongs to. */
export const fieldOf = (config: ElementConfig): ElementField =>
  config.variant in EDGE_CONFIG_SCHEMAS ? "edges" : "nodes";
