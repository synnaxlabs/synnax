// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { TimeRange } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/layout";
import { Range } from "@/range";
import { RangeServices } from "@/range/services";
import { renderLinkHook } from "@/testUtils";

describe("RangeServices.useLink", () => {
  it("should add, activate, and place the retrieved range", async () => {
    const key = "range-1";
    const retrieve = vi.fn(async () => ({
      key,
      name: "Burn Test",
      timeRange: TimeRange.ZERO,
    }));
    const client = { ranges: { retrieve } } as unknown as Client;
    const { handler, store } = renderLinkHook(RangeServices.useLink, {
      [Range.SLICE_NAME]: Range.reducer,
    });
    await handler({ client, key });
    expect(retrieve).toHaveBeenCalledWith(key);
    const state = store.getState();
    expect(Range.selectActiveKey(state)).toBe(key);
    expect(Range.select(state, key)?.name).toBe("Burn Test");
    expect(Layout.select(state, key)?.name).toBe("Burn Test");
  });
});
