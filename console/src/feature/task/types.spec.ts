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
    ["ethercat_read", "EtherCAT Read Task"],
    ["http_scan", "HTTP Scan Task"],
    ["labjack_read", "LabJack Read Task"],
    ["modbus_read", "Modbus Read Task"],
    ["ni_analog_read", "NI Analog Read Task"],
    ["opc_read", "OPC UA Read Task"],
    ["pagerduty_alert", "PagerDuty Alert Task"],
  ])("should map the vendor prefix of %s to %s", (type, expected) => {
    expect(Task.parseType(type)).toBe(expected);
  });

  it("should capitalize an unprefixed single-word type", () => {
    expect(Task.parseType("sequence")).toBe("Sequence Task");
  });

  it("should capitalize every word of an unprefixed multi-word type", () => {
    expect(Task.parseType("my_custom_type")).toBe("My Custom Type Task");
  });
});

describe("getIcon", () => {
  it.each([
    ["ethercat_read", Icon.Logo.EtherCAT],
    ["http_scan", Icon.Logo.HTTP],
    ["labjack_read", Icon.Logo.LabJack],
    ["modbus_read", Icon.Logo.Modbus],
    ["ni_analog_read", Icon.Logo.NI],
    ["opc_read", Icon.Logo.OPC],
    ["pagerduty_alert", Icon.Logo.PagerDuty],
  ])("should resolve the vendor icon for %s", (type, expected) => {
    expect(Task.getIcon(type).type).toBe(expected);
  });

  it("should fall back to the generic task icon for unknown types", () => {
    expect(Task.getIcon("something_else").type).toBe(Icon.Task);
  });
});
