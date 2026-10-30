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
import {
  banks,
  claims,
  groups,
  terminal,
  usedCount,
} from "@/feature/labjack/task/ports";
import {
  createAnalogReadChannel,
  createDigitalReadChannel,
} from "@/feature/labjack/testutil";

const { PORTS, T4_MODEL, T7_MODEL, AI_PORT_TYPE, DI_PORT_TYPE } = LabJack.Device;

const summary = (ports: LabJack.Device.Port[]) =>
  banks(ports).map(({ prefix, ports }) => [
    prefix,
    terminal(ports[0]),
    terminal(ports[ports.length - 1]),
  ]);

describe("ports", () => {
  describe("banks", () => {
    it("should split the ports where the terminal prefix changes", () => {
      expect(summary(PORTS[T4_MODEL][DI_PORT_TYPE])).toEqual([
        ["FIO", "FIO4", "FIO7"],
        ["EIO", "EIO0", "EIO7"],
        ["CIO", "CIO0", "CIO3"],
        ["MIO", "MIO0", "MIO2"],
      ]);
    });

    it("should name an analog input by the alias of its terminal", () => {
      expect(summary(PORTS[T4_MODEL][AI_PORT_TYPE])).toEqual([
        ["AIN", "AIN0", "AIN3"],
        ["FIO", "FIO4", "FIO7"],
        ["EIO", "EIO0", "EIO3"],
      ]);
    });

    it("should split the ports where the numbers skip", () => {
      expect(summary(PORTS[T7_MODEL][AI_PORT_TYPE])).toEqual([
        ["AIN", "AIN0", "AIN13"],
        ["AIN", "AIN48", "AIN127"],
      ]);
    });
  });

  describe("groups", () => {
    const [ain, fio] = banks(PORTS[T4_MODEL][AI_PORT_TYPE]);
    const nameOf = (key: string) => `chan_${key}`;

    it("should color each port by how many channels hold it", () => {
      const first = createAnalogReadChannel("AIN0");
      const second = createAnalogReadChannel("AIN1");
      const third = createAnalogReadChannel("AIN1");
      const [{ cells }] = groups({
        bank: ain,
        claims: claims([first, second, third]),
        selected: [],
        describe: nameOf,
      });
      expect(cells.map(({ state }) => state)).toEqual([
        "used",
        "conflict",
        "free",
        "free",
      ]);
      expect(cells[0].channel).toBe(first.key);
      expect(cells[0].tooltip).toBe(`AIN0: chan_${first.key}`);
      expect(cells[1].tooltip).toBe(
        `AIN1 is used by chan_${second.key} and chan_${third.key}`,
      );
      expect(cells[2].tooltip).toBe("AIN2 is free");
    });

    it("should label each port by its terminal number and key it by its port", () => {
      const [{ cells }] = groups({
        bank: fio,
        claims: new Map(),
        selected: [],
        describe: nameOf,
      });
      expect(cells.map(({ key, name, labels }) => [key, name, labels])).toEqual([
        ["AIN4", "FIO4", ["4"]],
        ["AIN5", "FIO5", ["5"]],
        ["AIN6", "FIO6", ["6"]],
        ["AIN7", "FIO7", ["7"]],
      ]);
    });

    it("should mark the port of a selected channel", () => {
      const ch = createAnalogReadChannel("AIN2");
      const [{ cells }] = groups({
        bank: ain,
        claims: claims([ch]),
        selected: [ch.key],
        describe: nameOf,
      });
      expect(cells.map(({ selected }) => selected)).toEqual([
        false,
        false,
        true,
        false,
      ]);
    });

    it("should lay a bank out eight ports to a row and sixteen to a group", () => {
      const [, mux] = banks(PORTS[T7_MODEL][AI_PORT_TYPE]);
      const laid = groups({
        bank: mux,
        claims: new Map(),
        selected: [],
        describe: nameOf,
      });
      expect(laid.map(({ columns, cells }) => [columns, cells.length])).toEqual([
        [8, 16],
        [8, 16],
        [8, 16],
        [8, 16],
        [8, 16],
      ]);
      expect(laid[4].cells.at(-1)).toMatchObject({ name: "AIN127", row: 2, column: 8 });
    });

    it("should narrow the group of a short bank to its length", () => {
      const [{ columns }] = groups({
        bank: fio,
        claims: new Map(),
        selected: [],
        describe: nameOf,
      });
      expect(columns).toBe(4);
    });
  });

  describe("usedCount", () => {
    it("should count the ports that hold a channel", () => {
      const channels = [
        createAnalogReadChannel("AIN0"),
        createAnalogReadChannel("AIN0"),
        createDigitalReadChannel("DIO4"),
      ];
      expect(usedCount(PORTS[T4_MODEL][AI_PORT_TYPE], claims(channels))).toBe(1);
    });
  });
});
