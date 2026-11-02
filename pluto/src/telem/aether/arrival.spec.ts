// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SyntheticCore } from "@synnaxlabs/client/testutil";
import { Rate, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterEach, describe, expect, it, vi } from "vitest";

import { createFactory } from "@/telem/aether/factory";
import { sourcePipeline } from "@/telem/aether/pipeline";
import { streamChannelValue } from "@/telem/aether/remote";
import { type BooleanSource } from "@/telem/aether/telem";
import { telemTest } from "@/telem/aether/testutil";
import { withinBounds } from "@/telem/aether/transformers";

const START = TimeStamp.seconds(1_000_000);

describe("boolean symbol source on a channel with no data", () => {
  const closers: (() => Promise<void>)[] = [];
  afterEach(async () => {
    await Promise.all(closers.splice(0).map(async (close) => await close()));
  });

  // The Core has written nothing. Its first sample, sin(2), falls inside the bounds.
  const setup = async () => {
    const core = new SyntheticCore({ rate: Rate.hz(1000), start: START, now: START });
    const client = telemTest.createClient(core);
    const source = createFactory(client).create(
      sourcePipeline("boolean", {
        connections: [{ from: "valueStream", to: "threshold" }],
        segments: {
          valueStream: streamChannelValue({ channel: core.data[0].key }),
          threshold: withinBounds({ trueBound: { lower: 0.9, upper: 1.1 } }),
        },
        outlet: "threshold",
      }),
    ) as BooleanSource;
    closers.push(async () => {
      source.cleanup?.();
      await client.feed.close();
    });
    const handler = vi.fn();
    source.onChange(handler);
    await expect.poll(() => source.value().status.variant).toBe("success");
    return { core, source, handler };
  };

  it("should not report an arrival when the stream registers", async () => {
    const { handler } = await setup();
    expect(handler).not.toHaveBeenCalled();
  });

  it("should still read as empty after the stream registers", async () => {
    const { source } = await setup();
    expect(source.value().data).toBe(false);
  });

  it("should report an arrival once real data lands", async () => {
    const { core, source, handler } = await setup();
    core.advance(TimeSpan.milliseconds(1));
    await expect.poll(() => handler.mock.calls.length).toBe(1);
    expect(source.value().data).toBe(true);
  });
});
