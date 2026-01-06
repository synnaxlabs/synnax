// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import type * as v0 from "@/version/types/v0";

export const sliceStateZ = z.object({
  version: z.literal("1.0.0"),
  consoleVersion: z.string(),
  updateNotificationsSilenced: z.boolean(),
});
export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  version: "1.0.0",
  consoleVersion: "0.0.0",
  updateNotificationsSilenced: false,
};

export const migrate: (state: v0.SliceState) => SliceState = (state) => ({
  version: "1.0.0",
  consoleVersion: state.version,
  updateNotificationsSilenced: false,
});
