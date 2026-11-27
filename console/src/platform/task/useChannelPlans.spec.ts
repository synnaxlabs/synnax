// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, holdRequests } from "@synnaxlabs/client/testutil";
import { TimeStamp } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { createConsoleWrapper, createTestStore } from "@/testutil";

const client = createTestClient();
const MISSING_KEY = 4294967295;

describe("useChannelPlans", () => {
  it("should keep the plan of an unchanged source until a new plan resolves", async () => {
    const store = await createTestStore();
    const { wrapper } = await createConsoleWrapper({ client, store });
    const { result, rerender } = renderHook(
      ({ b }: { b: string }) =>
        Task.useChannelPlans([{ a: "a" }, { b }], (planner) =>
          b === "b"
            ? ["plan a", "plan b"]
            : ["plan a2", planner.retrieve(MISSING_KEY)?.name ?? "plan c"],
        ),
      { wrapper, initialProps: { b: "b" } },
    );
    await waitFor(() => expect(result.current).toEqual(["plan a", "plan b"]));
    const { release } = holdRequests(client, "/channel/retrieve");
    rerender({ b: "c" });
    expect(result.current).toEqual(["plan a", undefined]);
    act(() => release());
    await waitFor(() => expect(result.current).toEqual(["plan a2", "plan c"]));
  });

  it("should plan again when a timestamp in a source changes", async () => {
    const store = await createTestStore();
    const { wrapper } = await createConsoleWrapper({ client, store });
    const { result, rerender } = renderHook(
      ({ at }: { at: TimeStamp }) =>
        Task.useChannelPlans([{ default: at }], () => [at.toString()]),
      { wrapper, initialProps: { at: TimeStamp.seconds(1) } },
    );
    await waitFor(() =>
      expect(result.current).toEqual([TimeStamp.seconds(1).toString()]),
    );
    rerender({ at: TimeStamp.seconds(2) });
    await waitFor(() =>
      expect(result.current).toEqual([TimeStamp.seconds(2).toString()]),
    );
  });

  it("should stop a plan whose sources change while it loads", async () => {
    const store = await createTestStore();
    const { wrapper } = await createConsoleWrapper({ client, store });
    const { release } = holdRequests(client, "/channel/retrieve");
    const runs: string[] = [];
    const { result, rerender } = renderHook(
      ({ b }: { b: string }) =>
        Task.useChannelPlans([{ b }], (planner) => {
          runs.push(b);
          return [planner.retrieve(MISSING_KEY)?.name ?? b];
        }),
      { wrapper, initialProps: { b: "b" } },
    );
    await waitFor(() => expect(runs).toEqual(["b"]));
    rerender({ b: "c" });
    await waitFor(() => expect(runs).toEqual(["b", "c"]));
    act(() => release());
    await waitFor(() => expect(result.current).toEqual(["c"]));
    await act(async () => {});
    expect(runs).toEqual(["b", "c", "c"]);
  });
});
