// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

const createField = (
  name: string,
  type: icd.JSONField["type"],
  rest: Partial<icd.JSONField> = {},
): icd.JSONField => icd.jsonFieldZ.parse({ name, pointer: `/${name}`, type, ...rest });

const onChannel = (field: icd.JSONField, channel = 0): icd.ChannelWriteBinding => ({
  type: "channel",
  field: field.key,
  channel,
});

describe("ICD.Write.suit", () => {
  it("should put each field with no default on a channel, and leave the rest unbound", () => {
    const setpoint = createField("setpoint", "float");
    const mode = createField("mode", "string", { default: "auto" });
    expect(ICD.Write.suit([], [setpoint, mode])).toEqual([onChannel(setpoint)]);
  });

  it("should put the first field on a channel when every field has a default", () => {
    const mode = createField("mode", "string", { default: "auto" });
    const open = createField("open", "boolean", { default: false });
    expect(ICD.Write.suit([], [mode, open])).toEqual([onChannel(mode)]);
  });

  it("should put the first field on a channel when only fixed values remain", () => {
    const mode = createField("mode", "string", { default: "auto" });
    const open = createField("open", "boolean", { default: false });
    const fixed: icd.WriteBinding = {
      type: "value",
      field: open.key,
      value: { type: "boolean", value: true },
    };
    expect(ICD.Write.suit([fixed], [mode, open])).toEqual([onChannel(mode), fixed]);
  });

  it("should keep the bindings that suit their fields", () => {
    const setpoint = createField("setpoint", "float");
    const mode = createField("mode", "string");
    const bindings: icd.WriteBinding[] = [
      { ...onChannel(setpoint, 12), dataType: "int32" },
      { type: "value", field: mode.key, value: { type: "string", value: "auto" } },
    ];
    expect(ICD.Write.suit(bindings, [setpoint, mode])).toEqual(bindings);
  });

  it("should drop the binding of a field that the message no longer has", () => {
    const setpoint = createField("setpoint", "float");
    const gone = createField("gone", "float");
    expect(
      ICD.Write.suit([onChannel(setpoint, 12), onChannel(gone, 13)], [setpoint]),
    ).toEqual([onChannel(setpoint, 12)]);
  });

  it("should put a field on a channel when its fixed value is of another kind", () => {
    const mode = createField("mode", "integer");
    const bindings: icd.WriteBinding[] = [
      { type: "value", field: mode.key, value: { type: "string", value: "auto" } },
    ];
    expect(ICD.Write.suit(bindings, [mode])).toEqual([onChannel(mode)]);
  });

  it("should drop a data type that a scaled field does not take", () => {
    const count = createField("count", "integer", { scale: 0.5 });
    expect(
      ICD.Write.suit([{ ...onChannel(count, 12), dataType: "int64" }], [count]),
    ).toEqual([onChannel(count, 12)]);
  });
});

describe("ICD.Write.commandName", () => {
  it("should name the command channel of a field after the prefix and the field", () => {
    const field = icd.jsonFieldZ.parse({
      name: "pump speed",
      pointer: "/speed",
      type: "float",
    });
    expect(ICD.Write.commandName("plant_pumps", field)).toBe(
      "plant_pumps_pump_speed_cmd",
    );
  });
});

describe("ICD.Write.unbind", () => {
  it("should clear the channel of each channel binding and keep fixed values", () => {
    const setpoint = createField("setpoint", "float");
    const mode = createField("mode", "string");
    const message: icd.WriteMessage = {
      message: "8d6a1ec4-7c8e-4f0a-9b8e-0c7f0d5c1a02",
      bindings: [
        onChannel(setpoint, 12),
        { type: "value", field: mode.key, value: { type: "string", value: "auto" } },
      ],
    };
    expect(ICD.Write.unbind(message)).toEqual({
      ...message,
      bindings: [onChannel(setpoint), message.bindings[1]],
    });
  });
});
