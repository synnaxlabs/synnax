// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Icon } from "@/icon";

const SHIPPED_KEYS = [
  "line-plot",
  "schematic",
  "task",
  "arc",
  "device",
  "range",
  "bar-chart",
  "pie-chart",
  "table",
  "terminal",
  "waveform",
  "pulse",
  "dashboard",
  "map",
  "database",
  "valve",
  "pump",
  "tank",
  "barrel",
  "gas-cylinder",
  "air",
  "rack",
  "power",
  "cable",
  "pressure",
  "temperature",
  "force",
  "torque",
  "strain",
  "acceleration",
  "velocity",
  "voltage",
  "current",
  "resistance",
  "rocket",
  "satellite",
  "airplane",
  "flame",
  "flask",
  "microscope",
  "robot-arm",
  "wrench",
  "gear",
  "timer",
  "warning",
  "danger",
  "bell",
  "shield",
  "lock",
  "first-aid",
  "checklist",
  "flag",
  "target",
  "eye",
  "factory",
  "building",
  "globe",
  "people",
  "sensor",
  "drop",
  "snowflake",
  "sun",
  "lightbulb",
  "star",
];

describe("pickable", () => {
  describe("PICKABLE", () => {
    it("should keep every shipped key, since stored data refers to them", () => {
      const keys = Icon.PICKABLE.map(({ key }) => key);
      expect(keys).toEqual(expect.arrayContaining(SHIPPED_KEYS));
    });

    it("should give every icon a unique key", () => {
      const keys = Icon.PICKABLE.map(({ key }) => key);
      expect(new Set(keys).size).toEqual(keys.length);
    });

    it("should use lowercase kebab-case keys", () => {
      Icon.PICKABLE.forEach(({ key }) => expect(key).toMatch(/^[a-z]+(-[a-z]+)*$/));
    });
  });

  describe("getPickable", () => {
    it("should return the icon with the key", () => {
      expect(Icon.getPickable("valve")?.name).toEqual("Valve");
    });

    it("should return undefined for an unknown key", () => {
      expect(Icon.getPickable("not-an-icon")).toBeUndefined();
    });
  });
});
