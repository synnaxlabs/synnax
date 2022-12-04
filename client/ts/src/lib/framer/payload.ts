import { z } from 'zod';
import { ChannelPayload } from '../channel/payload';

import { DataType, TimeRange, TimeStamp, TypedArray } from '../telem';

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
  data: z.string().transform(
    (s) =>
      new Uint8Array(
        atob(s)
          .split('')
          .map((c) => c.charCodeAt(0))
      )
  ),
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
      const typedArr = record[ch?.key || ''];
      return {
        dataType: ch.dataType,
        data: new Uint8Array(typedArr?.buffer),
      };
    }),
  };
};
