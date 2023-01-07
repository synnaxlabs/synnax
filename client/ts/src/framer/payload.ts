// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { DataType, TArray, TimeRange, TimeStamp } from "../telem";

export const arrayPayloadSchema = z.object({
  timeRange: z
    .object({
      start: z.number().transform((n) => new TimeStamp(n)),
      end: z.number().transform((n) => new TimeStamp(n)),
    })
    .transform((o) => new TimeRange(o.start, o.end))
    .optional(),
  dataType: z.string().transform((s) => new DataType(s)),
  data: z.string().transform((s) => Buffer.from(s, "base64").buffer),
});

export type ArrayPayload = z.infer<typeof arrayPayloadSchema>;

export const framePayloadSchema = z.object({
  keys: z.string().array().default([]),
  arrays: arrayPayloadSchema.array().default([]),
});

export type FramePayload = z.infer<typeof framePayloadSchema>;

export const arrayFromPayload = (payload: ArrayPayload): TArray => {
  const { dataType, data } = payload;
  return new TArray(data, dataType);
};

export const arrayToPayload = (array: TArray): ArrayPayload => {
  return {
    timeRange: array._timeRange,
    dataType: array.dataType,
    data: array.buffer,
  };
};
