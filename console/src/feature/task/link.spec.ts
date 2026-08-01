// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { assert, describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { Task } from "@/feature/task";
import { createSelectedPanel } from "@/platform/task/testutil";
import { assertDefined, createConsoleWrapper } from "@/testutil";

const client = createTestClient();

describe("Task.useLink", () => {
  it("should open a task tab for the retrieved task", async () => {
    const rack = await client.racks.create({ name: `test-rack-${id.create()}` });
    const task = await rack.createTask({
      name: "Analog Read",
      type: NI.Task.ANALOG_READ_TYPE,
      config: {},
    });
    const { wrapper, store } = await createConsoleWrapper({ client });
    const created = await createSelectedPanel(store, client);
    // useOpenTab reads the panel query cache; warm it and keep it subscribed
    // so dispatches stay visible.
    await client.panels.retrieve(created.panelKey);
    const { result } = renderHook(() => Task.useLink(), { wrapper });
    await act(async () => {
      await result.current({ client, key: String(task.key) });
    });
    await waitFor(async () => {
      const doc = await client.panels.retrieve(created.panelKey);
      assert(doc.root.variant === "leaf", "panel root is not a leaf");
      const tab = doc.root.tabs.find(
        (t): t is panel.TabView =>
          t.variant === "view" && t.type === NI.Task.ANALOG_READ_TYPE,
      );
      assertDefined(tab, "task tab was not opened");
      expect(tab.args).toEqual({ taskKey: task.key });
    });
  });
});
