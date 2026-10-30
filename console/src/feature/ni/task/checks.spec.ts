// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ni } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { NI } from "@/feature/ni";
import {
  analogReadHardwareZ,
  counterReadHardwareZ,
  digitalHardwareZ,
} from "@/feature/ni/task/checks";
import { createHardware } from "@/feature/ni/task/testutil";

const device: NI.Device.Device = {
  key: "dev",
  rack: 1,
  location: "Dev1",
  make: "NI",
  model: "PCIe-6363",
  name: "dev",
  configured: true,
  properties: { ...NI.Device.ZERO_PROPERTIES, hardware: createHardware() },
};

const aiChannel = (
  key: string,
  port: number,
  overrides: Partial<NI.Task.AIChannel> = {},
): NI.Task.AIChannel =>
  ({
    ...NI.Task.createAIChannel(),
    key,
    port,
    device: device.key,
    ...overrides,
  }) as NI.Task.AIChannel;

const analogRead = (
  channels: NI.Task.AIChannel[],
  sampleRate = 10,
): ni.AnalogReadConfig => ({
  ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
  sampleRate,
  channels,
});

const issues = (schema: z.ZodType, value: unknown) => {
  const result = schema.safeParse(value);
  if (result.success) return [];
  return result.error.issues.map(({ path, message, ...rest }) => ({
    path,
    message,
    variant: "params" in rest ? rest.params?.variant : undefined,
  }));
};

describe("analogReadHardwareZ", () => {
  it("should pass channels on free ports that take them", () => {
    const config = analogRead([aiChannel("a", 0), aiChannel("b", 1)]);
    expect(issues(analogReadHardwareZ([device]), config)).toEqual([]);
  });

  it("should pass a device with no port list", () => {
    const bare = { ...device, properties: NI.Device.ZERO_PROPERTIES };
    const config = analogRead([aiChannel("a", 99)]);
    expect(issues(analogReadHardwareZ([bare]), config)).toEqual([]);
  });

  it("should mark a port the device does not have", () => {
    expect(
      issues(analogReadHardwareZ([device]), analogRead([aiChannel("a", 20)])),
    ).toEqual([
      { path: ["channels", 0, "port"], message: "The device has no port ai20" },
    ]);
  });

  it("should mark a type the port does not take", () => {
    const config = analogRead([
      {
        ...NI.Task.createAIChannel("ai_thermocouple"),
        key: "a",
        port: 8,
        device: "dev",
      },
    ] as NI.Task.AIChannel[]);
    expect(issues(analogReadHardwareZ([device]), config)).toEqual([
      {
        path: ["channels", 0, "type"],
        message: "ai8 does not take thermocouple channels",
      },
    ]);
  });

  it("should mark a terminal configuration the port does not take", () => {
    const config = analogRead([aiChannel("a", 8, { terminalConfig: "Diff" })]);
    expect(issues(analogReadHardwareZ([device]), config)).toEqual([
      {
        path: ["channels", 0, "terminalConfig"],
        message: "ai8 does not take differential",
      },
    ]);
  });

  it("should mark a channel on the negative input of a differential channel", () => {
    const config = analogRead([
      aiChannel("a", 0),
      aiChannel("b", 8, { terminalConfig: "RSE" }),
    ]);
    expect(issues(analogReadHardwareZ([device]), config)).toEqual([
      {
        path: ["channels", 0, "port"],
        message: "Another channel uses ai8, its negative input",
      },
      {
        path: ["channels", 1, "port"],
        message: "ai8 is the negative input of another channel",
      },
    ]);
  });

  it("should mark a sample rate above the single channel maximum as an error", () => {
    const config = analogRead([aiChannel("a", 0)], 3_000_000);
    expect(issues(analogReadHardwareZ([device]), config)).toEqual([
      { path: ["sampleRate"], message: "PCIe-6363 samples at most 2,000,000 Hz" },
    ]);
  });

  it("should warn of a sample rate above the multiplexed maximum", () => {
    const config = analogRead([aiChannel("a", 0), aiChannel("b", 1)], 600_000);
    expect(issues(analogReadHardwareZ([device]), config)).toEqual([
      {
        path: ["sampleRate"],
        message: "PCIe-6363 may not sample 2 channels faster than 500,000 Hz",
        variant: "warning",
      },
    ]);
  });

  it("should count the extra scan channels of a thermocouple device", () => {
    const thermocouple = {
      ...NI.Task.createAIChannel("ai_thermocouple"),
      key: "a",
      port: 0,
      device: "dev",
    } as NI.Task.AIChannel;
    const config = analogRead([thermocouple], 400_000);
    expect(issues(analogReadHardwareZ([device]), config)).toEqual([
      {
        path: ["sampleRate"],
        message: "PCIe-6363 may not sample 3 channels faster than 333,333 Hz",
        variant: "warning",
      },
    ]);
  });
});

describe("counterReadHardwareZ", () => {
  it("should mark a counter the device does not have", () => {
    const config = {
      ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
      channels: [{ ...NI.Task.createCIChannel(), key: "a", port: 4, device: "dev" }],
    };
    expect(issues(counterReadHardwareZ([device]), config)).toEqual([
      { path: ["channels", 0, "port"], message: "The device has no counter ctr4" },
    ]);
  });
});

describe("digitalHardwareZ", () => {
  const line = (key: string, port: number, l: number): NI.Task.DIChannel => ({
    ...NI.Task.createDIChannel(),
    key,
    port,
    line: l,
  });

  it("should mark a line past the width of its port", () => {
    expect(
      issues(digitalHardwareZ(device, "input"), { channels: [line("a", 1, 8)] }),
    ).toEqual([
      { path: ["channels", 0, "line"], message: "port1 has 8 lines, numbered 0 to 7" },
    ]);
  });

  it("should mark both channels on one line", () => {
    expect(
      issues(digitalHardwareZ(device, "input"), {
        channels: [line("a", 0, 2), line("b", 0, 2)],
      }),
    ).toEqual([
      { path: ["channels", 0, "line"], message: "Another channel uses port0/line2" },
      { path: ["channels", 1, "line"], message: "Another channel uses port0/line2" },
    ]);
  });
});
