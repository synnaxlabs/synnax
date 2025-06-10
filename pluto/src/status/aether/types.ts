// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional, status, TimeStamp, unknownRecordZ } from "@synnaxlabs/x";
import { z } from "zod";

export const specZ = z.object({
  key: z.string(),
  variant: status.variantZ,
  message: z.string(),
  description: z.string().optional(),
  time: TimeStamp.z,
  data: unknownRecordZ.optional(),
});
export interface Spec extends z.infer<typeof specZ> {}

export interface CrudeSpec extends Optional<Spec, "time" | "key"> {}
