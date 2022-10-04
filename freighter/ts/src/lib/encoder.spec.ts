import test from 'ava';
import { z } from 'zod';

import { ENCODERS } from './encoder';

const SampleSchema = z.object({
  channelKey: z.string(),
  timeStamp: z.number(),
  value: z.unknown(),
});

ENCODERS.forEach((encoder) => {
  test(`[encoder] - encoder ${encoder.contentType}`, (t) => {
    const sample = {
      channelKey: 'test',
      timeStamp: 123,
      value: new Uint8Array([1, 2, 3]),
    };
    const encoded = encoder.encode(sample);
    const decoded = encoder.decode(encoded, SampleSchema);
    t.deepEqual(decoded, decoded);
  });
});
