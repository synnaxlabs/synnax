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

import { NI } from "@/feature/ni";
import { createTestDevice } from "@/platform/device/testutil";
import { createWithHook } from "@/platform/task/testutil";

const client = createTestClient();

describe("NI.Task create", () => {
  it("should keep the channels of the given analog read config", async () => {
    const config = NI.Task.analogReadConfigZ.parse({ autoStart: true });
    const tsk = await createWithHook(
      NI.Task.useCreateAnalogRead,
      NI.Task.ANALOG_READ_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels).toEqual([]);
  });

  it("should create an analog write task on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      NI.Task.useCreateAnalogWrite,
      NI.Task.ANALOG_WRITE_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.config.device).toBe(dev.key);
    expect(tsk.config.channels).toHaveLength(1);
  });

  it("should keep the channels of the given analog write config", async () => {
    const config = NI.Task.analogWriteConfigZ.parse({ autoStart: true });
    const tsk = await createWithHook(
      NI.Task.useCreateAnalogWrite,
      NI.Task.ANALOG_WRITE_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels).toEqual([]);
  });

  it("should keep the channels of the given counter read config", async () => {
    const config = NI.Task.counterReadConfigZ.parse({ autoStart: true });
    const tsk = await createWithHook(
      NI.Task.useCreateCounterRead,
      NI.Task.COUNTER_READ_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels).toEqual([]);
  });

  it("should create a digital read task on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      NI.Task.useCreateDigitalRead,
      NI.Task.DIGITAL_READ_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.config.device).toBe(dev.key);
    expect(tsk.config.channels).toHaveLength(1);
  });

  it("should keep the channels of the given digital read config", async () => {
    const config = NI.Task.digitalReadConfigZ.parse({ autoStart: true });
    const tsk = await createWithHook(
      NI.Task.useCreateDigitalRead,
      NI.Task.DIGITAL_READ_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels).toEqual([]);
  });

  it("should create a digital write task on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      NI.Task.useCreateDigitalWrite,
      NI.Task.DIGITAL_WRITE_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.config.device).toBe(dev.key);
    expect(tsk.config.channels).toHaveLength(1);
  });

  it("should keep the channels of the given digital write config", async () => {
    const config = NI.Task.digitalWriteConfigZ.parse({ autoStart: true });
    const tsk = await createWithHook(
      NI.Task.useCreateDigitalWrite,
      NI.Task.DIGITAL_WRITE_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels).toEqual([]);
  });
});
