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

import { LabJack } from "@/feature/labjack";
import { createTestDevice } from "@/platform/device/testutil";
import { createWithHook } from "@/platform/task/testutil";

const client = createTestClient();

describe("LabJack.Task create", () => {
  it("should create a read task on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      LabJack.Task.useCreateRead,
      LabJack.Task.READ_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.config.device).toBe(dev.key);
  });

  it("should keep the channels of the given read config", async () => {
    const config = LabJack.Task.READ_SCHEMAS.config.parse({
      channels: [{ ...LabJack.Task.createReadChannel("analog"), key: "kept" }],
    });
    const tsk = await createWithHook(
      LabJack.Task.useCreateRead,
      LabJack.Task.READ_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels.map(({ key }) => key)).toEqual(["kept"]);
  });

  it("should create a write task with one channel on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      LabJack.Task.useCreateWrite,
      LabJack.Task.WRITE_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.config.device).toBe(dev.key);
    expect(tsk.config.channels).toHaveLength(1);
  });

  it("should keep the channels of the given write config", async () => {
    const config = LabJack.Task.WRITE_SCHEMAS.config.parse({
      channels: [{ ...LabJack.Task.createWriteChannel("digital"), key: "kept" }],
    });
    const tsk = await createWithHook(
      LabJack.Task.useCreateWrite,
      LabJack.Task.WRITE_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels.map(({ key }) => key)).toEqual(["kept"]);
  });
});
