import test from 'ava';

import { ENCODERS } from './encoder';

ENCODERS.forEach((encoder) => {
  test(`encoder ${encoder.contentType}`, (t) => {
    const sample = {
      channelKey: 'test',
      timeStamp: 123,
      value: new Uint8Array([1, 2, 3]),
    };
    const encoded = encoder.encode(sample);
    const decoded = encoder.decode(encoded);
    t.deepEqual(decoded, decoded);
  });
});
