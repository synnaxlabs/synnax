// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { z } from "zod";

import { Label } from "@/schematic/node/common/label";
import { telem } from "@/telem/aether";
import { Staleness } from "@/vis/staleness";

export const stateMappingZ = z.object({
  key: z.string(),
  name: z.string(),
  value: z.number(),
  color: color.crudeZ.optional(),
});
export type StateMapping = z.infer<typeof stateMappingZ>;

export const VARIANT = "stateIndicator" as const;

export const configZ = Label.labeledConfigZ.extend({
  variant: z.literal(VARIANT),
  source: telem.numberSourceSpecZ.optional(),
  color: color.crudeZ.optional(),
  inlineSize: z.number().optional(),
  options: z.array(stateMappingZ),
  ...Staleness.configZ.shape,
});
export type Config = z.infer<typeof configZ>;
