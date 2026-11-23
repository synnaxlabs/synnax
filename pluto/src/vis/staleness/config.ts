// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { staleness } from "@/vis/staleness/aether";

/**
 * configZ is the staleness timeout of a symbol config, the key {@link TimeoutField}
 * writes. It is optional, because an older symbol does not carry it.
 */
export const configZ = z.object({
  // Positive matches the aether schema, so the form rejects a timeout the worker would
  // rather than pushing it across and failing there.
  stalenessTimeout: z.number().positive().optional(),
});
export interface Config extends z.infer<typeof configZ> {}

/** The staleness a symbol starts with. */
export const ZERO_CONFIG: Config = { stalenessTimeout: staleness.DEFAULT_TIMEOUT };
