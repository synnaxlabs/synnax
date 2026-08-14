// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, text } from "@synnaxlabs/x";
import { z } from "zod";

import { Label } from "@/schematic/node/common/label";
import { telem } from "@/telem/aether";
import { Staleness } from "@/vis/staleness";

export const VARIANT = "stringDisplay" as const;

export const configZ = Label.labeledConfigZ.extend({
  variant: z.literal(VARIANT),
  color: color.crudeZ.optional(),
  textColor: color.crudeZ.optional(),
  tooltip: z.array(z.string()).optional(),
  inlineSize: z.number().optional(),
  telem: telem.stringSourceSpecZ.optional(),
  level: text.levelZ.optional(),
  ...Staleness.configZ.shape,
});
export type Config = z.infer<typeof configZ>;
