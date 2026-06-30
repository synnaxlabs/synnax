// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Layout } from "@/layout";
import { Range } from "@/range";
import { RangeServices } from "@/range/services";
import { renderLinkHook } from "@/testUtils";

const client = createTestClient();

describe("RangeServices.useLink", () => {
  it("should add, activate, and place the retrieved range", async () => {
    const range = await client.ranges.create({
      name: "Burn Test",
      timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
    });
    const { handler, store } = renderLinkHook(RangeServices.useLink, {
      [Range.SLICE_NAME]: Range.reducer,
    });
    await handler({ client, key: range.key });
    const state = store.getState();
    expect(Range.selectSelectedKey(state)).toBe(range.key);
    expect(Range.select(state, range.key)?.name).toBe("Burn Test");
    expect(Layout.select(state, range.key)?.name).toBe("Burn Test");
  });
});
