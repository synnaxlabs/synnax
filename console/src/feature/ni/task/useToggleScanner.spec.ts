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

import { NI } from "@/feature/ni";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const createNIRackWithScanner = async () => {
  const rack = await client.racks.create({
    name: uniqueName("ni_rack"),
    integrations: ["ni"],
  });
  const scanTask = await rack.createTask(
    {
      name: uniqueName("ni_scanner"),
      type: NI.Task.SCAN_TYPE,
      config: { enabled: true },
    },
    NI.Task.SCAN_SCHEMAS,
  );
  return { rack, scanTask };
};

describe("useToggleScanner", () => {
  it("should flip the enabled flag of the scanner task on the given rack", async () => {
    const { rack, scanTask } = await createNIRackWithScanner();
    const { wrapper } = await createConsoleWrapper({ client });
    const { result } = renderHook(() => NI.Task.useToggleScanner(rack.key), {
      wrapper,
    });
    await act(async () => {
      result.current();
    });
    await waitFor(async () => {
      const after = await client.tasks.retrieve({
        key: scanTask.key,
        schemas: NI.Task.SCAN_SCHEMAS,
      });
      expect(after.config.enabled).toBe(!scanTask.config.enabled);
    });
  });

  it("should not affect the scanner task on a different rack", async () => {
    const { rack: rackA, scanTask: taskA } = await createNIRackWithScanner();
    const { scanTask: taskB } = await createNIRackWithScanner();
    const { wrapper } = await createConsoleWrapper({ client });
    const { result } = renderHook(() => NI.Task.useToggleScanner(rackA.key), {
      wrapper,
    });
    await act(async () => {
      result.current();
    });
    await waitFor(async () => {
      const afterA = await client.tasks.retrieve({
        key: taskA.key,
        schemas: NI.Task.SCAN_SCHEMAS,
      });
      expect(afterA.config.enabled).toBe(!taskA.config.enabled);
    });
    const afterB = await client.tasks.retrieve({
      key: taskB.key,
      schemas: NI.Task.SCAN_SCHEMAS,
    });
    expect(afterB.config.enabled).toBe(taskB.config.enabled);
  });
});
