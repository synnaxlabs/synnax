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

describe("useChannelPlans", () => {
  it("should keep the plan of an unchanged source until a new plan resolves", async () => {
    const store = await createTestStore();
    const { wrapper } = await createConsoleWrapper({ client, store });
    let resolve: (plans: string[]) => void = () => {};
    const next = new Promise<string[]>((r) => {
      resolve = r;
    });
    const { result, rerender } = renderHook(
      ({ b }: { b: string }) =>
        Task.useChannelPlans([{ a: "a" }, { b }], async () =>
          b === "b" ? ["plan a", "plan b"] : await next,
        ),
      { wrapper, initialProps: { b: "b" } },
    );
    await waitFor(() => expect(result.current).toEqual(["plan a", "plan b"]));
    rerender({ b: "c" });
    expect(result.current).toEqual(["plan a", undefined]);
    act(() => resolve(["plan a2", "plan c"]));
    await waitFor(() => expect(result.current).toEqual(["plan a2", "plan c"]));
  });
});
