// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, resolveFocusedTab, selectTestProject } from "@/testutil";

const client = createTestClient();

describe("Range.useAddToNewPlot", () => {
  it("should plot the ranges without adding them to the favorites", async () => {
    const a = await createTestRange(client);
    const b = await createTestRange(client);
    const { wrapper, store } = await createConsoleWrapper({ client });
    await selectTestProject(store, client);
    const { result } = renderHook(() => Range.useAddToNewPlot(), { wrapper });
    act(() => result.current([a.key, b.key]));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    await waitFor(async () =>
      expect((await client.lineplots.retrieve(tab.resource.key)).ranges).toEqual([
        { variant: "key", key: a.key },
        { variant: "key", key: b.key },
      ]),
    );
    expect(store.getState()[Session.Range.SLICE_NAME].ranges).toEqual([]);
  });
});
