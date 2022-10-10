import { z } from 'zod';

import { TimeStamp } from '../telem';

export const SegmentPayloadSchema = z.object({
  channelKey: z.string(),
  start: z.number().transform((n) => new TimeStamp(n)),
  data: z.string().transform(
    (s) =>
      new Uint8Array(
        atob(s)
          .split('')
          .map((c) => c.charCodeAt(0))
      )
  ),
});

export type SegmentPayload = z.infer<typeof SegmentPayloadSchema>;
