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
import { createHardware, createNIDevice } from "@/feature/ni/task/testutil";
import { createTestDevice } from "@/platform/device/testutil";
import { createWithHook } from "@/platform/task/testutil";

const client = createTestClient();

// A module whose inputs and outputs are on separate ports, as on the NI 9375.
const createDIOModule = async (inputPort: number, outputPort: number) =>
  await createNIDevice(client, {
    properties: {
      hardware: {
        ...createHardware(),
        digitalPorts: [
          { port: inputPort, lines: 16, input: true, output: false },
          { port: outputPort, lines: 16, input: false, output: true },
        ],
      },
    },
  });

describe("NI.Task create", () => {
  it("should keep the channels of the given analog read config", async () => {
    const config = NI.Task.analogReadConfigZ.parse({
      channels: [{ ...NI.Task.createAIChannel(), key: "kept" }],
    });
    const tsk = await createWithHook(
      NI.Task.useCreateAnalogRead,
      NI.Task.ANALOG_READ_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels.map(({ key }) => key)).toEqual(["kept"]);
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
    const config = NI.Task.analogWriteConfigZ.parse({
      channels: [{ ...NI.Task.createAOChannel(), key: "kept" }],
    });
    const tsk = await createWithHook(
      NI.Task.useCreateAnalogWrite,
      NI.Task.ANALOG_WRITE_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels.map(({ key }) => key)).toEqual(["kept"]);
  });

  it("should keep the channels of the given counter read config", async () => {
    const config = NI.Task.counterReadConfigZ.parse({
      channels: [{ ...NI.Task.createCIChannel(), key: "kept" }],
    });
    const tsk = await createWithHook(
      NI.Task.useCreateCounterRead,
      NI.Task.COUNTER_READ_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels.map(({ key }) => key)).toEqual(["kept"]);
  });

  it("should start a digital read task on an input line", async () => {
    const dev = await createDIOModule(1, 0);
    const tsk = await createWithHook(
      NI.Task.useCreateDigitalRead,
      NI.Task.DIGITAL_READ_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.config.device).toBe(dev.key);
    expect(tsk.config.channels.map(({ port, line }) => [port, line])).toEqual([[1, 0]]);
  });

  it("should keep the channels of the given digital read config", async () => {
    const config = NI.Task.digitalReadConfigZ.parse({
      channels: [{ ...NI.Task.createDIChannel(), key: "kept" }],
    });
    const tsk = await createWithHook(
      NI.Task.useCreateDigitalRead,
      NI.Task.DIGITAL_READ_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels.map(({ key }) => key)).toEqual(["kept"]);
  });

  it("should start a digital write task on an output line", async () => {
    const dev = await createDIOModule(0, 1);
    const tsk = await createWithHook(
      NI.Task.useCreateDigitalWrite,
      NI.Task.DIGITAL_WRITE_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.config.device).toBe(dev.key);
    expect(tsk.config.channels.map(({ port, line }) => [port, line])).toEqual([[1, 0]]);
  });

  it("should keep the channels of the given digital write config", async () => {
    const config = NI.Task.digitalWriteConfigZ.parse({
      channels: [{ ...NI.Task.createDOChannel(), key: "kept" }],
    });
    const tsk = await createWithHook(
      NI.Task.useCreateDigitalWrite,
      NI.Task.DIGITAL_WRITE_SCHEMAS,
      { config },
    );
    expect(tsk.config.channels.map(({ key }) => key)).toEqual(["kept"]);
  });
});
