// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, location } from "@synnaxlabs/x";
import { z } from "zod";

import { Label } from "@/schematic/node/common/label";
import { text } from "@/text/base";

export const VARIANT = "offPageReference" as const;

export const configZ = z.object({
  variant: z.literal(VARIANT),
  orientation: location.outerZ.optional(),
  label: Label.configZ,
  level: text.levelZ.optional(),
  color: color.crudeZ.optional(),
  page: z.string().optional(),
  dblClickNav: z.boolean().optional(),
});
export type Config = z.infer<typeof configZ>;
