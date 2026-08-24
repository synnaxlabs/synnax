// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";

const ai = (key: string, device: string, port: number): NI.Task.AIChannel => {
  const channel = NI.Task.createAIChannel();
  if (!("port" in channel)) throw new Error("channel carries no port");
  channel.port = port;
  return { ...channel, key, device };
};

// The built-in temperature sensor has no port, so it never collides with one.
const aiTempBuiltin = (key: string, device: string): NI.Task.AIChannel => ({
  ...NI.Task.createAIChannel("ai_temp_builtin"),
  key,
  device,
});

const parseAnalog = (channels: NI.Task.AIChannel[]) =>
  NI.Task.deployAnalogReadConfigZ.safeParse({
    ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
    streamRate: 1000,
    sampleRate: 2000,
    channels,
  });

describe("port validation", () => {
  it("allows different ports on the same device", () => {
    expect(parseAnalog([ai("0", "dev1", 0), ai("1", "dev1", 1)]).success).toBe(true);
  });

  it("allows the same port on different devices", () => {
    expect(parseAnalog([ai("0", "dev1", 0), ai("1", "dev2", 0)]).success).toBe(true);
  });

  it("rejects a duplicate port on the same device", () => {
    expect(parseAnalog([ai("0", "dev1", 0), ai("1", "dev1", 0)]).success).toBe(false);
  });

  it("does not collide a portless channel with a real port", () => {
    expect(parseAnalog([ai("0", "dev1", 0), aiTempBuiltin("1", "dev1")]).success).toBe(
      true,
    );
  });

  it("enforces port uniqueness independently per device", () => {
    expect(parseAnalog([ai("0", "dev1", 3), ai("1", "dev2", 3)]).success).toBe(true);
    expect(parseAnalog([ai("0", "dev1", 3), ai("1", "dev1", 3)]).success).toBe(false);
  });
});
