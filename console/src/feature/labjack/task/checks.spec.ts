// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import { uniqueTerminalsZ } from "@/feature/labjack/task/checks";
import {
  createAnalogReadChannel,
  createAnalogWriteChannel,
  createDigitalReadChannel,
} from "@/feature/labjack/testutil";

const { T4_MODEL, T7_MODEL } = LabJack.Device;

const issues = (model: LabJack.Device.Model, channels: LabJack.Task.Channel[]) => {
  const result = uniqueTerminalsZ(model).safeParse({ channels });
  return result.success
    ? []
    : result.error.issues.map(({ path, message }) => [path, message]);
};

describe("uniqueTerminalsZ", () => {
  it("should pass channels on distinct terminals", () => {
    expect(
      issues(T4_MODEL, [
        createAnalogReadChannel("AIN0"),
        createDigitalReadChannel("DIO5"),
      ]),
    ).toEqual([]);
  });

  it("should flag each of two channels on one port", () => {
    expect(
      issues(T4_MODEL, [
        createAnalogWriteChannel("DAC0"),
        createAnalogWriteChannel("DAC1"),
        createAnalogWriteChannel("DAC0"),
      ]),
    ).toEqual([
      [["channels", 0, "port"], "Another channel uses DAC0"],
      [["channels", 2, "port"], "Another channel uses DAC0"],
    ]);
  });

  it.each([
    ["analog", [createAnalogReadChannel("AIN4"), createDigitalReadChannel("DIO4")]],
    ["digital", [createDigitalReadChannel("DIO4"), createAnalogReadChannel("AIN4")]],
  ])(
    "should flag ports of two types on one terminal, with the %s channel first",
    (_, channels) => {
      expect(issues(T4_MODEL, channels)).toEqual([
        [["channels", 0, "port"], "Another channel uses FIO4"],
        [["channels", 1, "port"], "Another channel uses FIO4"],
      ]);
    },
  );

  it("should pass the same ports on a model where they have their own terminals", () => {
    expect(
      issues(T7_MODEL, [
        createAnalogReadChannel("AIN4"),
        createDigitalReadChannel("DIO4"),
      ]),
    ).toEqual([]);
  });
});
