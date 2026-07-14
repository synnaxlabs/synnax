// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { Task } from "@/feature/task";
import { awaitTaskResourceTab, createSelectedPanel } from "@/platform/task/testutil";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

describe("Task.useLink", () => {
  it("should open the task's resource tab for the retrieved task", async () => {
    const rack = await client.racks.create({ name: `test-rack-${id.create()}` });
    const task = await rack.createTask({
      name: "Analog Read",
      type: NI.Task.ANALOG_READ_TYPE,
      config: {},
    });
    const { wrapper, store } = await createConsoleWrapper({ client });
    const created = await createSelectedPanel(wrapper, store, client);
    const { result } = renderHook(() => Task.useLink(), { wrapper });
    await act(async () => {
      await result.current({ client, key: String(task.key) });
    });
    expect(await awaitTaskResourceTab(created)).toBe(task.key);
  });
});
