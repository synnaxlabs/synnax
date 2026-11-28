// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { ICD } from "@/platform/icd";

const flags = vi.hoisted((): { textMessages: boolean } => ({ textMessages: true }));

vi.mock("@/flags", async (importOriginal) => {
  const { FLAGS } = await importOriginal<{ FLAGS: object }>();
  return {
    FLAGS: {
      ...FLAGS,
      get textMessages() {
        return flags.textMessages;
      },
    },
  };
});

const createBinaryField = (
  name: string,
  kind: icd.BinaryField["kind"],
  startBit: number,
): icd.BinaryField =>
  icd.binaryFieldZ.parse({ key: uuid.create(), name, kind, startBit, bitLength: 8 });

const createTextField = (name: string, position: number): icd.TextField =>
  icd.textFieldZ.parse({ key: uuid.create(), name, encoding: "delimited", position });

const createMessage = (payload: icd.Message["payload"]): icd.Message => ({
  key: uuid.create(),
  name: "command",
  payload,
});

const createField = (
  name: string,
  type: icd.JSONField["type"],
  rest: Partial<icd.JSONField> = {},
): icd.JSONField => icd.jsonFieldZ.parse({ name, pointer: `/${name}`, type, ...rest });

const onChannel = (
  field: { key: icd.FieldKey },
  channel = 0,
): icd.ChannelWriteBinding => ({
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

describe("ICD.Write.suit of text and binary fields", () => {
  it("should keep a fixed value of the kind of a binary field", () => {
    const valve = createBinaryField("valve", "unsigned", 0);
    const fixed: icd.WriteBinding = {
      type: "value",
      field: valve.key,
      value: { type: "unsigned", value: "1" },
    };
    const setpoint = createBinaryField("setpoint", "float", 8);
    expect(ICD.Write.suit([fixed], [valve, setpoint])).toEqual([
      fixed,
      onChannel(setpoint),
    ]);
  });

  it("should put a text field on a channel when its fixed value is not a float", () => {
    const level = createTextField("level", 1);
    const bindings: icd.WriteBinding[] = [
      { type: "value", field: level.key, value: { type: "integer", value: "3" } },
    ];
    expect(ICD.Write.suit(bindings, [level])).toEqual([onChannel(level)]);
  });

  it("should keep a numeric data type of a binary field and drop any other", () => {
    const count = createBinaryField("count", "unsigned", 0);
    const gear = createBinaryField("gear", "unsigned", 8);
    const kept: icd.WriteBinding = { ...onChannel(count, 12), dataType: "uint8" };
    expect(
      ICD.Write.suit(
        [kept, { ...onChannel(gear, 13), dataType: "string" }],
        [count, gear],
      ),
    ).toEqual([kept, onChannel(gear, 13)]);
  });
});

describe("ICD.Write.bindable", () => {
  it("should give each field of a text message", () => {
    const fields = [createTextField("level", 1), createTextField("flow", 2)];
    expect(
      ICD.Write.bindable(
        createMessage({ format: "text", delimiter: ",", prefix: "", fields }),
      ),
    ).toEqual(fields);
  });

  it("should leave out the identifier field of a binary message", () => {
    const id = createBinaryField("id", "unsigned", 0);
    const valve = createBinaryField("valve", "unsigned", 8);
    expect(
      ICD.Write.bindable(
        createMessage({
          format: "binary",
          identifier: { type: "field", field: id.key, value: 4 },
          fields: [id, valve],
        }),
      ),
    ).toEqual([valve]);
  });

  it("should give no fields of a text message while tasks take only JSON", () => {
    flags.textMessages = false;
    onTestFinished(() => {
      flags.textMessages = true;
    });
    const fields = [createTextField("level", 1)];
    expect(
      ICD.Write.bindable(
        createMessage({ format: "text", delimiter: ",", prefix: "", fields }),
      ),
    ).toBeNull();
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

describe("ICD.Write.addMessages", () => {
  const client = createTestClient();

  it("should send the time of each write from a timestamp field with no default", async () => {
    const time = createField("time", "timestamp", { timeFormat: "unix_ms" });
    const setpoint = createField("setpoint", "float");
    const { icd: key, messages } = await ICD.Write.addMessages({
      client,
      formats: new Map(),
      targets: [],
      name: uniqueName("valves"),
      messages: [{ name: "command", fields: [time, setpoint] }],
    });
    onTestFinished(async () => await client.icds.delete(key));
    const payload = {
      format: "json",
      fields: [{ ...time, default: { type: "now" } }, setpoint],
    };
    expect(messages[0].payload).toEqual(payload);
    const [stored] = (await client.icds.retrieve({ key })).messages;
    expect(stored.payload).toEqual(payload);
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
