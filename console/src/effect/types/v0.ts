// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { effect } from "@synnaxlabs/client";
import z from "zod/v4";

export const sliceStateZ = z.object({
  version: z.literal("0.0.0"),
  effects: z.record(z.string(), effect.effectZ),
});

export type Effect = z.infer<typeof effect.effectZ>;

export const ZERO_EFFECT: Effect = {
  key: "",
  name: "",
  enabled: false,
  slate: "",
};

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  effects: {},
};
