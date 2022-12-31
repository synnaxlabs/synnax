// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { ChannelPayload } from "../channel/payload";
import { DataType, TimeRange, TimeStamp, TypedArray } from "../telem";

export const arrayPayloadSchema = z.object({
  timeRange: z
    .object({
      start: z.number().transform((n) => new TimeStamp(n)),
      end: z.number().transform((n) => new TimeStamp(n)),
    })
    .transform((o) => new TimeRange(o.start, o.end))
    .optional(),
  dataType: z
    .string()
    .transform((s) => new DataType(s))
    .optional(),
  data: z.string().transform((s) => {
    const buf = new SharedArrayBuffer(s.length);
    const view = new Uint8Array(buf);
    view.set(
      s.split("").map((c) => c.charCodeAt(0)),
      0
    );
    return buf as ArrayBufferLike;
  }),
});

export type ArrayPayload = z.infer<typeof arrayPayloadSchema>;

export const framePayloadSchema = z.object({
  keys: z.string().array().nullable(),
  arrays: arrayPayloadSchema.array().nullable(),
});

export type FramePayload = z.infer<typeof framePayloadSchema>;

export const frameFromRecord = (
  channels: ChannelPayload[],
  record: Record<string, TypedArray>
): FramePayload => {
  return {
    keys: Object.keys(record),
    arrays: channels.map((ch) => {
      const typedArr = record[ch?.key ?? ""];
      return {
        dataType: ch.dataType,
        data: typedArr.buffer,
      };
    }),
  };
};
