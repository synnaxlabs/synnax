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

import { Task } from "@/platform/task";
import { createConsoleWrapper, createTestStore } from "@/testutil";

const client = createTestClient();

describe("useChannelPlan", () => {
  it("should return no plan for new inputs until their plan resolves", async () => {
    const store = await createTestStore();
    const { wrapper } = await createConsoleWrapper({ client, store });
    let resolve: (plan: string) => void = () => {};
    const planB = new Promise<string>((r) => {
      resolve = r;
    });
    const { result, rerender } = renderHook(
      ({ name }: { name: string }) =>
        Task.useChannelPlan({ name }, async () =>
          name === "a" ? "plan a" : await planB,
        ),
      { wrapper, initialProps: { name: "a" } },
    );
    await waitFor(() => expect(result.current).toBe("plan a"));
    rerender({ name: "b" });
    expect(result.current).toBeUndefined();
    act(() => resolve("plan b"));
    await waitFor(() => expect(result.current).toBe("plan b"));
  });
});
