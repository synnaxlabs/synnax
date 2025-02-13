// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

export const VARIANTS = [
  "success",
  "error",
  "warning",
  "info",
  "loading",
  "disabled",
  "secondary",
] as const;
export const variantZ = z.enum(VARIANTS);
export type Variant = z.infer<typeof variantZ>;

export const specZ = z.object({
  key: z.string(),
  variant: variantZ,
  message: z.string(),
  description: z.string().optional(),
  time: TimeStamp.z,
  data: z.record(z.unknown()).optional(),
});
export interface Spec extends z.infer<typeof specZ> {}

export interface CrudeSpec extends Optional<Spec, "time" | "key"> {}
