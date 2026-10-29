// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SyntheticCore } from "@synnaxlabs/client/testutil";
import { box, TimeStamp } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { renderHook } from "@/testutil/render";
import { Value } from "@/vis/value";
import { value } from "@/vis/value/aether";

const BOX = box.construct({ x: 0, y: 0 }, { width: 200, height: 50 });

describe("Value.use", () => {
  const closers: Array<() => Promise<void>> = [];
  afterEach(async () => {
    for (const close of closers.splice(0)) await close();
  });

  const connect = (): telem.Client => {
    const core = new SyntheticCore({
      rate: 10,
      start: TimeStamp.seconds(100),
      now: TimeStamp.seconds(110),
    });
    const client = telemTest.createClient(core);
    closers.push(async () => await client.feed.close());
    return client;
  };

  it("should return the problem of a failed read instead of toasting it", async () => {
    const { result, providers } = renderHook(
      () =>
        Value.use({
          aetherKey: "value",
          box: BOX,
          telem: telem.streamChannelStringValue({ channel: 999 }),
        }),
      {
        render: true,
        registry: value.REGISTRY,
        telem: { factories: [new telem.RemoteFactory(connect())] },
      },
    );
    expect(result.current.problems).toEqual([]);
    await waitFor(() =>
      expect(result.current.problems).toEqual([
        {
          variant: "error",
          name: "",
          message: "Failed to stream channel value",
          description: "channel 999 not found",
        },
      ]),
    );
    expect(providers.status?.state.statuses).toEqual([]);
  });
});
