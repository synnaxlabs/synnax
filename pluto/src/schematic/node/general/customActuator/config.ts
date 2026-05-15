// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x/color";
import { schematic } from "@synnaxlabs/client";

import { z } from "zod";

import { Toggle } from "@/schematic/node/common/toggle";

export const VARIANT = "customActuator" as const;

export const configZ = Toggle.toggleConfigZ.extend({
  variant: z.literal(VARIANT),
  specKey: z.string(),
  color: color.crudeZ.optional(),
  scale: z.number().optional(),
  stateOverrides: z.array(schematic.symbol.stateZ).optional(),
});
export type Config = z.infer<typeof configZ>;
