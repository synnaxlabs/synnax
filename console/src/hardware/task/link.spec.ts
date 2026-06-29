// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { describe, expect, it, vi } from "vitest";

import { NI } from "@/hardware/ni";
import { Task } from "@/hardware/task";
import { Layout } from "@/layout";
import { renderLinkHook } from "@/testUtils";

describe("Task.useLink", () => {
  it("should place a task layout for the retrieved task", async () => {
    const key = "task-1";
    const retrieve = vi.fn(async () => ({
      key,
      name: "Analog Read",
      type: NI.Task.ANALOG_READ_TYPE,
      snapshot: false,
    }));
    const client = { tasks: { retrieve } } as unknown as Client;
    const { handler, store } = renderLinkHook(Task.useLink);
    await handler({ client, key });
    expect(retrieve).toHaveBeenCalledWith({ key });
    expect(Layout.select(store.getState(), key)?.name).toBe("Analog Read");
  });
});
