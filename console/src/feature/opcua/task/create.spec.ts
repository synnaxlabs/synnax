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

import { OPCUA } from "@/feature/opcua";
import { createTestDevice } from "@/platform/device/testutil";
import { createWithHook } from "@/platform/task/testutil";

const client = createTestClient();

describe("OPCUA.Task create", () => {
  it("should create a read task on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      OPCUA.Task.useCreateRead,
      OPCUA.Task.READ_SCHEMAS,
      {
        deviceKey: dev.key,
      },
    );
    expect(tsk.name).toBe("OPC UA read task");
    expect(tsk.config.device).toBe(dev.key);
  });

  it("should create a read task with the given config", async () => {
    const config = OPCUA.Task.READ_SCHEMAS.config.parse({ sampleRate: 7 });
    const tsk = await createWithHook(
      OPCUA.Task.useCreateRead,
      OPCUA.Task.READ_SCHEMAS,
      {
        config,
      },
    );
    expect(tsk.config.sampleRate).toBe(7);
  });

  it("should create a write task on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      OPCUA.Task.useCreateWrite,
      OPCUA.Task.WRITE_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.name).toBe("OPC UA write task");
    expect(tsk.config.device).toBe(dev.key);
  });

  it("should create a write task with the given config", async () => {
    const config = OPCUA.Task.WRITE_SCHEMAS.config.parse({ autoStart: true });
    const tsk = await createWithHook(
      OPCUA.Task.useCreateWrite,
      OPCUA.Task.WRITE_SCHEMAS,
      { config },
    );
    expect(tsk.config.autoStart).toBe(true);
  });
});
