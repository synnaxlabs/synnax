import {encode} from "msgpackr";
import { describe, expect, test } from "vitest";
import { z } from "zod";

import { ENCODERS } from "./encoder";

const SampleSchema = z.object({
  channelKey: z.string(),
  timeStamp: z.number(),
  value: z.unknown(),
});

describe.each(ENCODERS)("encoder", (e) => {
  test(`[encoder] - encoder ${e.contentType} should encode correctly`, () => {
    const sample = {
      channelKey: "test",
      timeStamp: 123,
      value: new Array([1, 2, 3]),
    };
    const encoded = e.encode(sample);
    expect(e.decode(encoded, SampleSchema)).toEqual(sample);
  });
});
