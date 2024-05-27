// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { binary } from "@/binary";

const SampleSchema = z.object({
  channelKey: z.string(),
  timeStamp: z.number(),
  value: z.unknown(),
});

binary.ENCODERS.forEach((e) => {
  describe(`encoder ${e.contentType}`, () => {
    it("should correctly encode and decode items", () => {
      const sample = {
        channelKey: "test",
        timeStamp: 123,
        value: new Array([1, 2, 3]),
      };
      const encoded = e.encode(sample);
      expect(e.decode(encoded, SampleSchema)).toEqual(sample);
    });
  });
});

describe("JSON", () => {
  it("should correctly convert keys to snake case", () => {
    const sample = {
      channelKey: "test",
      timeStamp: 123,
      value: new Array([1, 2, 3]),
    };
    const encoded = binary.JSON_ECD.encodeString(sample);
    const parse = JSON.parse(encoded);
    expect(parse.channel_key).toEqual("test");
  });
  it("should correctly decode keys from snake case", () => {
    const sample = {
      channel_key: "test",
      time_stamp: 123,
      value: new Array([1, 2, 3]),
    };
    const encoded = JSON.stringify(sample);
    const decoded = binary.JSON_ECD.decodeString(encoded, SampleSchema);
    expect(decoded.channelKey).toEqual("test");
  });
});
