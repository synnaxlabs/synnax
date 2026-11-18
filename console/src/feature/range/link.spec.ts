// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import {
  renderLinkHook,
  resolveFocusedTab,
  selectRange,
  selectTestProject,
} from "@/testutil";

const client = createTestClient();

describe("Range.useLink", () => {
  it("should open the range without adding it to the favorites", async () => {
    const range = await client.ranges.create({
      name: "Burn Test",
      timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
    });
    const { handler, store } = await renderLinkHook(Range.useLink, { client });
    await selectTestProject(store, client);
    await handler({ client, key: range.key });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource).toEqual(ranger.resourceID(range.key));
    expect(selectRange(store.getState(), range.key)).toBeUndefined();
  });
});
