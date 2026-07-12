// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { Session } from "@/session";
import { renderLinkHook } from "@/testutil";

const client = createTestClient();

describe("Range.useLink", () => {
  it("should add and activate the retrieved range", async () => {
    const range = await client.ranges.create({
      name: "Burn Test",
      timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
    });
    const { handler, store } = await renderLinkHook(Range.useLink);
    await handler({ client, key: range.key });
    const state = store.getState();
    expect(Session.Range.selectSelectedKey(state)).toBe(range.key);
    expect(Session.Range.selectState(state, range.key)?.name).toBe("Burn Test");
  });
});
