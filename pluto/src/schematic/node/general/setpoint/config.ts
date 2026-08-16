// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, dimensions } from "@synnaxlabs/x";
import { z } from "zod";

import { size as componentSize } from "@/component/size";
import { Control } from "@/schematic/node/common/control";
import { Label } from "@/schematic/node/common/label";
import { telem } from "@/telem/aether";

export const VARIANT = "setpoint" as const;

export const configZ = Label.labeledConfigZ.extend({
  variant: z.literal(VARIANT),
  size: componentSize.optional(),
  sink: telem.numberSinkSpecZ.optional(),
  dimensions: dimensions.dimensionsZ.optional(),
  color: color.crudeZ.optional(),
  units: z.string().optional(),
  disabled: z.boolean().optional(),
  control: Control.stateConfigZ.optional(),
});
export type Config = z.infer<typeof configZ>;
