// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { NI } from "@/feature/ni";
import {
  analogReadHardwareZ,
  counterReadHardwareZ,
  digitalHardwareZ,
} from "@/feature/ni/task/checks";

const ai = (key: string, device: string, port: number): NI.Task.AIChannel => {
  const channel = NI.Task.createAIChannel();
  if (!("port" in channel)) throw new Error("channel carries no port");
  return { ...channel, key, device, port };
};

const issues = (schema: z.ZodType, value: unknown) => {
  const result = schema.safeParse(value);
  return result.success
    ? []
    : result.error.issues.map(({ path, message }) => ({ path, message }));
};

const analogRead = (channels: NI.Task.AIChannel[]) =>
  issues(analogReadHardwareZ([]), {
    ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
    channels,
  });

describe("analogReadHardwareZ", () => {
  it("should pass different ports on one device", () => {
    expect(analogRead([ai("a", "dev1", 0), ai("b", "dev1", 1)])).toEqual([]);
  });

  it("should pass one port on two devices", () => {
    expect(analogRead([ai("a", "dev1", 0), ai("b", "dev2", 0)])).toEqual([]);
  });

  it("should report each channel on a port another channel uses", () => {
    expect(analogRead([ai("a", "dev1", 3), ai("b", "dev1", 3)])).toEqual([
      { path: ["channels", 0, "port"], message: "Another channel uses ai3" },
      { path: ["channels", 1, "port"], message: "Another channel uses ai3" },
    ]);
  });

  it("should pass a port that only a disabled channel shares", () => {
    const disabled = { ...ai("b", "dev1", 3), disabled: true };
    expect(analogRead([ai("a", "dev1", 3), disabled])).toEqual([]);
  });

  it("should pass a channel with no port beside one on a port", () => {
    const builtin: NI.Task.AIChannel = {
      ...NI.Task.createAIChannel("ai_temp_builtin"),
      key: "b",
      device: "dev1",
    };
    expect(analogRead([ai("a", "dev1", 0), builtin])).toEqual([]);
  });
});

describe("counterReadHardwareZ", () => {
  const ci = (key: string, device: string, port: number): NI.Task.CIChannel => ({
    ...NI.Task.createCIChannel(),
    key,
    device,
    port,
  });
  const counterRead = (channels: NI.Task.CIChannel[]) =>
    issues(counterReadHardwareZ([]), {
      ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
      channels,
    });

  it("should report a counter two channels on one device share", () => {
    expect(counterRead([ci("a", "dev1", 0), ci("b", "dev1", 0)])).toHaveLength(2);
  });

  it("should pass one counter on two devices", () => {
    expect(counterRead([ci("a", "dev1", 0), ci("b", "dev2", 0)])).toEqual([]);
  });
});

describe("digitalHardwareZ", () => {
  it("should report a line two channels share on its line field", () => {
    const di = (key: string): NI.Task.DIChannel => ({
      ...NI.Task.createDIChannel(),
      key,
      port: 0,
      line: 2,
    });
    expect(
      issues(digitalHardwareZ(undefined, "input"), { channels: [di("a"), di("b")] }),
    ).toEqual([
      { path: ["channels", 0, "line"], message: "Another channel uses port0/line2" },
      { path: ["channels", 1, "line"], message: "Another channel uses port0/line2" },
    ]);
  });
});
