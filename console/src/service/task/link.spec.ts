// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NI } from "@/hardware/ni";
import { Task } from "@/hardware/task";
import { Layout } from "@/layout";
import { renderLinkHook } from "@/testUtils";

const client = createTestClient();

describe("Task.useLink", () => {
  it("should place a task layout for the retrieved task", async () => {
    const rack = await client.racks.create({ name: `test-rack-${id.create()}` });
    const task = await rack.createTask({
      name: "Analog Read",
      type: NI.Task.ANALOG_READ_TYPE,
      config: {},
    });
    const { handler, store } = renderLinkHook(Task.useLink);
    await handler({ client, key: String(task.key) });
    const placed = Layout.selectByFilter(
      store.getState(),
      (l) => l.name === "Analog Read",
    );
    expect(placed).toBeDefined();
  });
});
