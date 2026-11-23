// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { closedBounds } from "@synnaxlabs/x";
import { z } from "zod";

// Releases up to 0.62 took minVal and maxVal. A config that still sets them fails
// instead of quietly taking the default range.
const replacedZ = (key: string) =>
  z.never({ error: `${key} was replaced by range: set range: { lower, upper }` });

/** MinMaxVal bounds the expected signal range in scaled units. */
export const minMaxValZ = z.object({
  /** range is the expected signal range, in scaled units. */
  range: closedBounds.closedBoundsZ().prefault({ lower: 0, upper: 1 }),
  minVal: replacedZ("minVal").optional(),
  maxVal: replacedZ("maxVal").optional(),
});
export interface MinMaxVal extends z.infer<typeof minMaxValZ> {}
