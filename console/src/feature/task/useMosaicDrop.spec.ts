// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { Task } from "@/feature/task";
import { Session } from "@/session";
import { renderHookWithConsole, uniqueName } from "@/testutil";

const client = createTestClient();

const createTask = async () => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await rack.createTask({
    name: uniqueName("task"),
    type: NI.Task.ANALOG_READ_TYPE,
    config: {},
  });
};

describe("Task.useMosaicDrop", () => {
  it("should place the task layout in the target mosaic node", async () => {
    const t = await createTask();
    const { result, store } = await renderHookWithConsole(() => Task.useMosaicDrop(), {
      client,
    });
    result.current({ id: t.ontologyID, nodeKey: 7, location: "center" });
    await waitFor(() => {
      const placed = Session.Layout.select(store.getState(), t.key);
      expect(placed?.key).toBe(t.key);
      expect(placed?.tab).toMatchObject({ mosaicKey: 7, location: "center" });
    });
  });
});
