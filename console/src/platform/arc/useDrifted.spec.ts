// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, status, task } from "@synnaxlabs/client";
import { crdt } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/platform/arc";
import { client } from "@/platform/arc/testutil";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const createDeployed = async (): Promise<{ a: arc.Arc; tsk: task.Task }> => {
  const rck = await client.racks.create({ name: uniqueName("rack") });
  const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
  const tsk = await client.arcs.deploy(a.key, rck.key);
  if (tsk == null) throw new Error("expected a deployment task");
  return { a, tsk };
};

const setRunning = async (
  tsk: task.Task,
  overrides: { configHash?: string; rack?: number } = {},
): Promise<void> => {
  await client.statuses.set(
    status.create<ReturnType<typeof task.statusDetailsZ>>({
      key: task.statusKey(tsk.key),
      variant: "success",
      message: "Running",
      details: {
        task: tsk.key,
        running: true,
        cmd: "",
        configHash: overrides.configHash ?? tsk.configHash,
        rack: overrides.rack ?? tsk.rack,
        data: {},
      },
    }),
  );
};

const renderDrifted = async (a: arc.Arc) => {
  const { wrapper } = await createConsoleWrapper({ client });
  return renderHook(
    () => ({ drifted: Arc.useDrifted(a.key), task: Arc.useTask(a.key, a.name) }),
    { wrapper },
  );
};

const editProgram = async (key: arc.Key): Promise<void> => {
  const gen = new crdt.Text(2);
  const ops = gen.insert(0, "a -> b").map((op) => arc.insertChar(op));
  await client.arcs.dispatch(key, ops);
};

describe("arc useDrifted", () => {
  it("should not drift while the task is not running", async () => {
    const { a } = await createDeployed();
    await editProgram(a.key);
    const { result } = await renderDrifted(a);
    await waitFor(() => expect(result.current.task.taskKey).not.toBe(""));
    expect(result.current.drifted).toBe(false);
  });

  it("should not drift while the deployed instance matches", async () => {
    const { a, tsk } = await createDeployed();
    await setRunning(tsk);
    const { result } = await renderDrifted(a);
    await waitFor(() => expect(result.current.task.running).toBe(true));
    expect(result.current.drifted).toBe(false);
  });

  it("should drift when the program changes after deploy", async () => {
    const { a, tsk } = await createDeployed();
    await setRunning(tsk);
    const { result } = await renderDrifted(a);
    await waitFor(() => expect(result.current.task.running).toBe(true));
    await editProgram(a.key);
    await waitFor(() => expect(result.current.drifted).toBe(true));
  });

  it("should drift when the task moves to a different rack while running", async () => {
    const { a, tsk } = await createDeployed();
    await setRunning(tsk);
    const other = await client.racks.create({ name: uniqueName("rack") });
    await client.arcs.deploy(a.key, other.key);
    const { result } = await renderDrifted(a);
    await waitFor(() => expect(result.current.task.running).toBe(true));
    await waitFor(() => expect(result.current.drifted).toBe(true));
  });
});
