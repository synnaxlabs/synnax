// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x/telem";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Name = string;
export type Keys = Key[];
export type Names = Name[];
export type Params = Key | Name | Keys | Names;

export const payloadZ = z.object({
  key: keyZ,
  name: z.string().min(1),
  timeRange: TimeRange.z,
  color: z.string().optional(),
});
export type Payload = z.infer<typeof payloadZ>;

export const newPayloadZ = payloadZ.extend({
  key: z.string().uuid().optional(),
});
export type NewPayload = z.infer<typeof newPayloadZ>;

export type ParamAnalysisResult =
  | {
      single: true;
      variant: "keys";
      normalized: Keys;
      actual: Key;
    }
  | {
      single: true;
      variant: "names";
      normalized: Names;
      actual: Name;
    }
  | {
      single: false;
      variant: "keys";
      normalized: Keys;
      actual: Keys;
    }
  | {
      single: false;
      variant: "names";
      normalized: Names;
      actual: Names;
    };

export const analyzeParams = (params: Params): ParamAnalysisResult => {
  const normal = toArray(params) as Keys | Names;
  if (normal.length === 0) {
    throw new Error("Range params must not be empty");
  }
  const isKey = keyZ.safeParse(normal[0]).success;
  return {
    single: !Array.isArray(params),
    variant: isKey ? "keys" : "names",
    normalized: normal,
    actual: params,
  } as const as ParamAnalysisResult;
};
