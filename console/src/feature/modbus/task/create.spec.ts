// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";

import { Modbus } from "@/feature/modbus";
import { createTestDevice } from "@/platform/device/testutil";
import { createWithHook } from "@/platform/task/testutil";

const client = createTestClient();

describe("Modbus.Task create", () => {
  it("should create a read task with one channel on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      Modbus.Task.useCreateRead,
      Modbus.Task.READ_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.config.device).toBe(dev.key);
    expect(tsk.config.channels).toHaveLength(1);
  });

  it("should create a write task with one channel on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      Modbus.Task.useCreateWrite,
      Modbus.Task.WRITE_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.config.device).toBe(dev.key);
    expect(tsk.config.channels).toHaveLength(1);
  });
});
