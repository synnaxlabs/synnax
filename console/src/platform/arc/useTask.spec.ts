// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/platform/arc";
import { client, TIMEOUT } from "@/platform/arc/testutil";
import { createConsoleWrapper } from "@/testutil";

describe("arc useTask", () => {
  it("should report a not-deployed status for an arc with no task", async () => {
    const arc = await client.arcs.create({
      name: "Untasked Arc",
      mode: "graph",
      graph: { nodes: [], edges: [] },
    });
    const { wrapper } = await createConsoleWrapper({ client });
    const { result } = renderHook(() => Arc.useTask(arc.key, arc.name), { wrapper });
    await waitFor(
      () => expect(result.current.taskStatus.message).toBe("Not deployed yet"),
      TIMEOUT,
    );
    expect(result.current.running).toBe(false);
    expect(result.current.taskKey).toBe("");
  });

  it("should expose a callable start/stop handler that no-ops without a task", async () => {
    const arc = await client.arcs.create({
      name: "Untasked Arc 2",
      mode: "graph",
      graph: { nodes: [], edges: [] },
    });
    const { wrapper } = await createConsoleWrapper({ client });
    const { result } = renderHook(() => Arc.useTask(arc.key, arc.name), { wrapper });
    await waitFor(
      () => expect(result.current.taskStatus.message).toBe("Not deployed yet"),
      TIMEOUT,
    );
    expect(() => result.current.onStartStop()).not.toThrow();
  });
});
