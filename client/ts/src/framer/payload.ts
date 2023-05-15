// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, LazyArray, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

export const arrayPayloadSchema = z.object({
  timeRange: z
    .object({
      start: z.number().transform((n) => new TimeStamp(n)),
      end: z.number().transform((n) => new TimeStamp(n)),
    })
    .transform((o) => new TimeRange(o.start, o.end))
    .optional(),
  dataType: z.string().transform((s) => new DataType(s)),
  data: z.string().transform(
    (s) =>
      new Uint8Array(
        atob(s)
          .split("")
          .map((c) => c.charCodeAt(0))
      ).buffer
  ),
});

export type ArrayPayload = z.infer<typeof arrayPayloadSchema>;

export const framePayload = z.object({
  keys: z.string().array().nullable().default([]).or(z.number().array().nullable().default([])),
  arrays: arrayPayloadSchema.array().nullable().default([]),
});

export type FramePayload = z.infer<typeof framePayload>;

export const arrayFromPayload = (payload: ArrayPayload): LazyArray => {
  const { dataType, data, timeRange } = payload;
  return new LazyArray(data, dataType, timeRange);
};

export const arrayToPayload = (array: LazyArray): ArrayPayload => {
  return {
    timeRange: array._timeRange,
    dataType: array.dataType,
    data: new Uint8Array(array.data.buffer),
  };
};
