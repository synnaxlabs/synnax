// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mqtt } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";

const DEVICE_TAG: MQTT.Task.SparkplugTagID = {
  group: "plant",
  edgeNode: "line1",
  device: "ovenA",
  tag: "zone 1/temperature",
};

const NODE_TAG: MQTT.Task.SparkplugTagID = { ...DEVICE_TAG, device: "" };

describe("sparkplugPropertiesKey", () => {
  it("should join the IDs of a device tag under the Sparkplug B namespace", () => {
    expect(MQTT.Task.sparkplugPropertiesKey(DEVICE_TAG)).toBe(
      "spBv1.0/plant/line1/ovenA/zone 1/temperature",
    );
  });

  it("should keep an empty device level for a tag of the edge node", () => {
    expect(MQTT.Task.sparkplugPropertiesKey(NODE_TAG)).toBe(
      "spBv1.0/plant/line1//zone 1/temperature",
    );
  });

  it("should tell a device tag from a tag of the edge node with a slash", () => {
    const deviceTag = { ...NODE_TAG, device: "ovenA", tag: "temperature" };
    const nodeTag = { ...NODE_TAG, tag: "ovenA/temperature" };
    expect(MQTT.Task.sparkplugPropertiesKey(deviceTag)).not.toBe(
      MQTT.Task.sparkplugPropertiesKey(nodeTag),
    );
  });
});

describe("sparkplugChannelName", () => {
  it("should build a valid channel name from the device name and the IDs", () => {
    expect(MQTT.Task.sparkplugChannelName("My broker", DEVICE_TAG)).toBe(
      "My_broker_plant_line1_ovenA_zone_1_temperature",
    );
  });

  it("should leave the device level out for a tag of the edge node", () => {
    expect(MQTT.Task.sparkplugChannelName("broker", NODE_TAG)).toBe(
      "broker_plant_line1_zone_1_temperature",
    );
  });
});

describe("fromSparkplugDataType", () => {
  it.each<[MQTT.Task.SparkplugDataType, string]>([
    ["int8", "int8"],
    ["int16", "int16"],
    ["int32", "int32"],
    ["int64", "int64"],
    ["uint8", "uint8"],
    ["uint16", "uint16"],
    ["uint32", "uint32"],
    ["uint64", "uint64"],
    ["float", "float32"],
    ["double", "float64"],
    ["boolean", "uint8"],
    ["string", "string"],
    ["date_time", "timestamp"],
  ])("should hold %s values in a %s channel", (type, dataType) => {
    expect(MQTT.Task.fromSparkplugDataType(type)).toBe(dataType);
  });

  it("should cover every Sparkplug B data type of a task config", () => {
    mqtt.SPARKPLUG_DATA_TYPES.forEach((type) =>
      expect(MQTT.Task.fromSparkplugDataType(type)).toBeTypeOf("string"),
    );
  });
});

describe("toSparkplugDataType", () => {
  it.each(mqtt.SPARKPLUG_DATA_TYPES)("should keep the browsed type %s", (type) => {
    expect(MQTT.Task.toSparkplugDataType(type)).toBe(type);
  });

  it.each(["Text", "UUID"])("should read a %s tag as a string", (type) => {
    expect(MQTT.Task.toSparkplugDataType(type)).toBe("string");
  });

  it("should reject a data type that a task cannot take", () => {
    expect(() => MQTT.Task.toSparkplugDataType("DataSet")).toThrow(/Invalid option/);
  });
});
