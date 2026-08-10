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

import { DEFAULT_TIMEOUT } from "@/vis/staleness/aether/staleness";

/**
 * configZ carries the staleness keys a symbol config holds. Extend a symbol's schema
 * with its shape to pair the schema with {@link Fields}, which writes these exact keys.
 * Both are optional, because a symbol saved before it gained staleness config carries
 * neither.
 */
export const configZ = z.object({
  stalenessTimeout: z.number().optional(),
  stalenessColor: color.colorZ.optional(),
});
export interface Config extends z.infer<typeof configZ> {}

/** The staleness a symbol starts with. An unset color resolves to the theme warning. */
export const ZERO_CONFIG: Config = {
  stalenessTimeout: DEFAULT_TIMEOUT,
  stalenessColor: color.ZERO,
};
