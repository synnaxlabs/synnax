// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { allocSuite } from "@synnaxlabs/x/bench";
import { test } from "vitest";

import { createKeys, createPayload } from "@/framer/benchutil";
import { Frame } from "@/framer/frame";

const CHANNEL_COUNTS = [10, 100, 1000];

// Every streamed message constructs a Frame from the decoded payload.
test("from payload", async ({ bench }) => {
  await bench.compare(
    ...CHANNEL_COUNTS.map((n) => {
      const payload = createPayload(createKeys(n), 10);
      return bench(`${n}ch`, () => {
        new Frame(payload);
      });
    }),
  );
});

// get() scans every column, so per-key lookups over a whole frame are quadratic.
test("get", async ({ bench }) => {
  await bench.compare(
    ...CHANNEL_COUNTS.flatMap((n) => {
      const keys = createKeys(n);
      const frame = new Frame(createPayload(keys, 10));
      return [
        bench(`one key of ${n}ch`, () => {
          frame.get(keys[n - 1]);
        }),
        bench(`all keys of ${n}ch`, () => {
          for (const k of keys) frame.get(k);
        }),
      ];
    }),
  );
});

{
  const keys = createKeys(100);
  const payload = createPayload(keys, 10);
  const frame = new Frame(payload);
  allocSuite("frame 100ch x 10smp", [
    ["from payload", () => new Frame(payload)],
    ["get one key", () => void frame.get(keys[99])],
    [
      "get all keys",
      () => {
        for (const k of keys) frame.get(k);
      },
    ],
  ]);
}
