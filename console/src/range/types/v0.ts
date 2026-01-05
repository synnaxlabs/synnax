// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { numericTimeRangeZ, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

export const baseRangeZ = z.object({
  name: z.string(),
  key: z.string(),
  persisted: z.boolean(),
});

export const staticRangeZ = baseRangeZ.extend({
  variant: z.literal("static"),
  timeRange: numericTimeRangeZ,
});

export type StaticRange = z.infer<typeof staticRangeZ>;

export const dynamicRangeZ = baseRangeZ.extend({
  variant: z.literal("dynamic"),
  span: z.number(),
});

export type DynamicRange = z.infer<typeof dynamicRangeZ>;

export const rangeZ = z.union([staticRangeZ, dynamicRangeZ]);

export type Range = z.infer<typeof rangeZ>;

export const sliceStateZ = z.object({
  version: z.literal("0.0.0"),
  activeRange: z.string().nullable(),
  ranges: z.record(z.string(), rangeZ),
});

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  activeRange: null,
  ranges: {
    rolling30s: {
      key: "recent",
      variant: "dynamic",
      name: "Rolling 30s",
      span: Number(TimeSpan.seconds(30)),
      persisted: false,
    },
    rolling1m: {
      key: "rolling1m",
      variant: "dynamic",
      name: "Rolling 1m",
      span: Number(TimeSpan.minutes(1)),
      persisted: false,
    },
    rolling5m: {
      key: "rolling5m",
      variant: "dynamic",
      name: "Rolling 5m",
      span: Number(TimeSpan.minutes(5)),
      persisted: false,
    },
    rolling15m: {
      key: "rolling15m",
      variant: "dynamic",
      name: "Rolling 15m",
      span: Number(TimeSpan.minutes(15)),
      persisted: false,
    },
    rolling30m: {
      key: "rolling30m",
      variant: "dynamic",
      name: "Rolling 30m",
      span: Number(TimeSpan.minutes(30)),
      persisted: false,
    },
  },
};
