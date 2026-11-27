// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { Kafka } from "@/feature/kafka";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { MQTT } from "@/feature/mqtt";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { Sift } from "@/feature/sift";
import { Task } from "@/feature/task";

describe("parseType", () => {
  it.each([
    ["ethercat_read", "EtherCAT read task"],
    ["http_scan", "HTTP scan task"],
    ["labjack_read", "LabJack read task"],
    ["modbus_read", "Modbus read task"],
    ["mqtt_read", "MQTT read task"],
    ["ni_analog_read", "NI analog read task"],
    ["opc_read", "OPC UA read task"],
    ["pagerduty_alert", "PagerDuty alert task"],
  ])("should map the vendor prefix of %s to %s", (type, expected) => {
    expect(Task.parseType(type)).toBe(expected);
  });

  it("should capitalize the first word of an unprefixed type", () => {
    expect(Task.parseType("sequence")).toBe("Sequence task");
  });

  it("should sentence-case an unprefixed multi-word type", () => {
    expect(Task.parseType("my_custom_type")).toBe("My custom type task");
  });
});

// A rack runs a task only when it lists the integration that the task form checks.
describe("getIntegration", () => {
  it.each([
    ["ethercat_read", EtherCAT.Device.INTEGRATION_KEY],
    ["http_read", HTTP.Device.INTEGRATION_KEY],
    ["kafka_read", Kafka.Device.INTEGRATION_KEY],
    ["labjack_read", LabJack.Device.INTEGRATION_KEY],
    ["modbus_read", Modbus.Device.INTEGRATION_KEY],
    ["mqtt_read", MQTT.Device.INTEGRATION_KEY],
    ["ni_analog_read", NI.Device.INTEGRATION_KEY],
    ["opc_read", OPCUA.Device.INTEGRATION_KEY],
    ["sift_stream", Sift.Device.INTEGRATION],
  ])("should give %s the integration %s that its Driver lists", (type, key) => {
    expect(Task.getIntegration(type)?.prefix).toBe(key);
  });

  it("should give an Arc task no integration", () => {
    expect(Task.getIntegration("arc")).toBeUndefined();
  });
});

describe("getIcon", () => {
  it.each([
    ["ethercat_read", Icon.Logo.EtherCAT],
    ["http_scan", Icon.Logo.HTTP],
    ["labjack_read", Icon.Logo.LabJack],
    ["modbus_read", Icon.Logo.Modbus],
    ["mqtt_read", Icon.Logo.MQTT],
    ["ni_analog_read", Icon.Logo.NI],
    ["opc_read", Icon.Logo.OPCUA],
    ["pagerduty_alert", Icon.Logo.PagerDuty],
  ])("should resolve the vendor icon for %s", (type, expected) => {
    expect(Task.getIcon(type).type).toBe(expected);
  });

  it("should fall back to the generic task icon for unknown types", () => {
    expect(Task.getIcon("something_else").type).toBe(Icon.Task);
  });
});
