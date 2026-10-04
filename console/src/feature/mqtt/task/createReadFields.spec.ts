// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";

const summarize = (fields: MQTT.Task.ReadField[]) =>
  fields.map(({ pointer, dataType, disabled }) => ({ pointer, dataType, disabled }));

describe("createReadFields", () => {
  it("should create a float64 field at the pointer of each number", () => {
    const fields = MQTT.Task.createReadFields('{"temperature": 21.5, "pressure": 3}');
    expect(summarize(fields)).toEqual([
      { pointer: "/temperature", dataType: "float64", disabled: false },
      { pointer: "/pressure", dataType: "float64", disabled: false },
    ]);
  });

  it("should create a uint8 field for a boolean", () => {
    const fields = MQTT.Task.createReadFields('{"open": true}');
    expect(summarize(fields)).toEqual([
      { pointer: "/open", dataType: "uint8", disabled: false },
    ]);
  });

  it("should recurse into nested objects", () => {
    const fields = MQTT.Task.createReadFields('{"pump": {"outlet": {"flow": 1.2}}}');
    expect(summarize(fields)).toEqual([
      { pointer: "/pump/outlet/flow", dataType: "float64", disabled: false },
    ]);
  });

  it("should skip arrays and nulls", () => {
    const fields = MQTT.Task.createReadFields('{"samples": [1, 2], "error": null}');
    expect(fields).toEqual([]);
  });

  it("should escape the ~ and / characters of a member name", () => {
    const fields = MQTT.Task.createReadFields('{"flow/rate~avg": 1}');
    expect(fields[0].pointer).toBe("/flow~1rate~0avg");
  });

  it("should create one field that takes the whole payload for a bare scalar", () => {
    expect(summarize(MQTT.Task.createReadFields("23.4"))).toEqual([
      { pointer: "", dataType: "float64", disabled: false },
    ]);
  });

  it("should keep string fields enabled when the payload holds only strings", () => {
    const fields = MQTT.Task.createReadFields('{"state": "ON", "mode": "AUTO"}');
    expect(summarize(fields)).toEqual([
      { pointer: "/state", dataType: "string", disabled: false },
      { pointer: "/mode", dataType: "string", disabled: false },
    ]);
  });

  it("should disable string fields when the payload also holds numbers", () => {
    const fields = MQTT.Task.createReadFields('{"temperature": 21.5, "unit": "C"}');
    expect(summarize(fields)).toEqual([
      { pointer: "/temperature", dataType: "float64", disabled: false },
      { pointer: "/unit", dataType: "string", disabled: true },
    ]);
  });

  it("should give the fields of one payload distinct keys", () => {
    const fields = MQTT.Task.createReadFields('{"a": 1, "b": 2}');
    expect(new Set(fields.map(({ key }) => key)).size).toBe(2);
  });

  it.each(["", "not json", '{"temperature": 21.'])(
    "should create no fields for the sample %j",
    (payload) => {
      expect(MQTT.Task.createReadFields(payload)).toEqual([]);
    },
  );
});
