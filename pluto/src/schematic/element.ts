// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod";

import { Edge } from "@/schematic/edge";
import { Node } from "@/schematic/node";

export const elementConfigZ = z.discriminatedUnion("variant", [
  ...Node.configZ.options,
  ...Edge.configZ.options,
]);
export type ElementConfig = z.infer<typeof elementConfigZ>;

export const ELEMENT_REGISTRY = { ...Node.REGISTRY, ...Edge.REGISTRY };
