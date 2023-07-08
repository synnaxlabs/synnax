// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, toArray } from "@synnaxlabs/x";
import { z } from "zod";

export const rangeKey = z.string().uuid();
export type RangeKey = z.infer<typeof rangeKey>;
export type RangeName = string;
export type RangeKeys = RangeKey[];
export type RangeNames = RangeName[];
export type RangeParams = RangeKey | RangeName | RangeKeys | RangeNames;

export const rangePayload = z.object({
  key: z.string().uuid(),
  name: z.string(),
  timeRange: TimeRange.z,
});
export type RangePayload = z.infer<typeof rangePayload>;

export const newRangePayload = rangePayload.extend({
  key: z.string().uuid().optional(),
});
export type NewRangePayload = z.infer<typeof newRangePayload>;

export type RangeParamAnalysisResult =
  | {
      single: true;
      variant: "keys";
      normalized: RangeKeys;
      actual: RangeKey;
    }
  | {
      single: true;
      variant: "names";
      normalized: RangeNames;
      actual: RangeName;
    }
  | {
      single: false;
      variant: "keys";
      normalized: RangeKeys;
      actual: RangeKeys;
    }
  | {
      single: false;
      variant: "names";
      normalized: RangeNames;
      actual: RangeNames;
    };

export const analyzeRangeParams = (params: RangeParams): RangeParamAnalysisResult => {
  const normal = toArray(params) as RangeKeys | RangeNames;
  if (normal.length === 0) {
    throw new Error("Range params must not be empty");
  }
  const isKey = rangeKey.safeParse(normal[0]).success;
  return {
    single: !Array.isArray(params),
    variant: isKey ? "keys" : "names",
    normalized: normal,
    actual: params,
  } as const as RangeParamAnalysisResult;
};
