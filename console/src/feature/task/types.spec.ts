// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { describe, expect, it } from "vitest";

import { Task } from "@/feature/task";

describe("parseType", () => {
  it.each([
    ["ethercat_read", "EtherCAT read task"],
    ["http_scan", "HTTP scan task"],
    ["labjack_read", "LabJack read task"],
    ["modbus_read", "Modbus read task"],
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

describe("getIcon", () => {
  it.each([
    ["ethercat_read", Icon.Logo.EtherCAT],
    ["http_scan", Icon.Logo.HTTP],
    ["labjack_read", Icon.Logo.LabJack],
    ["modbus_read", Icon.Logo.Modbus],
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
