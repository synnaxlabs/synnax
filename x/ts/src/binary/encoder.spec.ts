// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";
import { z } from "zod";

import { binary } from "@/binary";

const SampleSchema = z.object({
  channelKey: z.string(),
  timeStamp: z.number(),
  value: z.unknown(),
});

describe.each(binary.ENCODERS)("encoder", (e) => {
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
