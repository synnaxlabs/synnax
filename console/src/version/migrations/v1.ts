// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import * as v0 from "@/version/migrations/v0";

export const sliceStateZ = z.object({
  version: z.literal("1.0.0"),
  consoleVersion: z.string(),
  silenced: z.boolean(),
});
export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  version: "1.0.0",
  consoleVersion: "0.0.0",
  silenced: false,
};

export const migrate: (v: v0.SliceState) => SliceState = (v) => ({
  version: "1.0.0",
  consoleVersion: v.version,
  silenced: false,
});
