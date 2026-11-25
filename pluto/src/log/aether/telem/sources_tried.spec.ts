// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type framer, ValidationError } from "@synnaxlabs/client";
import { SyntheticCore } from "@synnaxlabs/client/testutil";
import { Unreachable } from "@synnaxlabs/freighter";
import { Rate, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterEach, describe, expect, it } from "vitest";

import { StreamMultiChannelLog } from "@/log/aether/telem/sources";
import { type telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";

const RATE = Rate.hz(10);
const START = TimeStamp.seconds(1_000_000);
const HISTORY = 300;
const NOW = START.add(TimeSpan.seconds(HISTORY / 10));
const POLL = { timeout: 5000, interval: 5 };
const OPTIONS: Partial<framer.FeedProps> = {
  batchDebounce: TimeSpan.milliseconds(5),
  removalDelay: TimeSpan.ZERO,
  breaker: { baseInterval: TimeSpan.milliseconds(10) },
};

describe("StreamMultiChannelLog breaker", () => {
  const closers: Array<() => Promise<void> | void> = [];
  afterEach(async () => {
    for (const close of closers.splice(0).reverse()) await close();
  });

  const setup = (fail: () => Error | null) => {
    const core = new SyntheticCore({ rate: RATE, start: START, now: NOW, channels: 2 });
    const base = telemTest.createClient(core, OPTIONS);
    closers.push(async () => await base.feed.close());
    const client: telem.Client = {
      ...base,
      calculations: {
        outputs: {
          retrieve: async (keys) => {
            const err = fail();
            if (err != null) throw err;
            return await base.calculations.outputs.retrieve(keys);
          },
        },
      },
    };
    const [a] = core.data.map((c) => c.key);
    const log = new StreamMultiChannelLog(
      client,
      { channels: [a] },
      undefined,
      core.now,
      undefined,
      OPTIONS.breaker,
      TimeSpan.milliseconds(5),
    );
    closers.push(() => log.cleanup());
    return log;
  };

  it("should show the channel after the output lookup fails once to reach the Core", async () => {
    let failed = false;
    const log = setup(() => {
      if (failed) return null;
      failed = true;
      return new Unreachable();
    });
    await expect.poll(() => log.value().data.length, POLL).toBe(HISTORY);
  });

  it("should report a definitive output lookup failure as an error status", async () => {
    const log = setup(() => new ValidationError("bad output lookup"));
    await expect
      .poll(() => log.value().status, POLL)
      .toMatchObject({ variant: "error", description: "bad output lookup" });
  });
});
