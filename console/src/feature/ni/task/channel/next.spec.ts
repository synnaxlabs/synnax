// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ni } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { Channel } from "@/feature/ni/task/channel";

// AI channels are a discriminated union and the built-in temperature sensor variant
// carries no port, so the specs reach for one through these.
const portOf = (channel: NI.Task.AIChannel): number | undefined =>
  "port" in channel ? channel.port : undefined;

const aiChannel = (
  key: string,
  port: number,
  type: NI.Task.AIChannelType = "ai_voltage",
  extra: { channel?: number } = {},
): NI.Task.AIChannel => {
  const channel = NI.Task.createAIChannel(type);
  if (!("port" in channel)) throw new Error(`${type} carries no port`);
  channel.port = port;
  return { ...channel, key, ...extra };
};

describe("Channel", () => {
  describe("Channel.createNextDI", () => {
    it("should create a new DI channel with line 0 when no channels exist", () => {
      const channels: NI.Task.DIChannel[] = [];
      const result = Channel.createNextDI(channels);
      expect(result).toEqual({
        channel: 0,
        key: expect.any(String),
        line: 0,
        port: 0,
        name: "",
        disabled: false,
      });
    });
    it("should create a new DI channel with the next available line number", () => {
      const channels: NI.Task.DIChannel[] = [
        { ...NI.Task.createDIChannel(), key: "1", line: 0, channel: 3 },
        { ...NI.Task.createDIChannel(), key: "2", line: 1, channel: 4 },
      ];
      const result = Channel.createNextDI(channels);
      expect(result.line).toBe(2);
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.channel).toBe(0);
    });
    it("should take a line on the port of the copied channel without port data", () => {
      const channels: NI.Task.DIChannel[] = [
        { ...NI.Task.createDIChannel(), key: "1", port: 1, line: 0 },
        { ...NI.Task.createDIChannel(), key: "2", port: 2, line: 0 },
      ];
      const result = Channel.createNextDI(channels, { copy: "1" });
      expect(result).toMatchObject({ port: 1, line: 1 });
    });
  });

  describe("Channel.createNextDO", () => {
    it("should create a new DO channel with line 0 when no channels exist", () => {
      const channels: NI.Task.DOChannel[] = [];
      const result = Channel.createNextDO(channels);
      expect(result).toEqual({
        cmdChannel: 0,
        stateChannel: 0,
        key: expect.any(String),
        line: 0,
        port: 0,
        cmdChannelName: "",
        stateChannelName: "",
        disabled: false,
      });
    });

    it("should create a new DO channel with the next available line number", () => {
      const channels: NI.Task.DOChannel[] = [
        { ...NI.Task.createDOChannel(), key: "1", line: 0 },
        { ...NI.Task.createDOChannel(), key: "2", line: 1 },
      ];
      const result = Channel.createNextDO(channels);
      expect(result.line).toBe(2);
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
    });
  });

  describe("Channel.createNextAI", () => {
    it("should create a new AI channel with port 0 when no channels exist", () => {
      const channels: NI.Task.AIChannel[] = [];
      const result = Channel.createNextAI(channels);
      expect(portOf(result)).toBe(0);
      expect(result.key).toBeDefined();
      expect(result.channel).toBe(0);
    });

    it("should create a new AI channel with the next available port number", () => {
      const channels = [aiChannel("1", 0), aiChannel("2", 1)];
      const result = Channel.createNextAI(channels);
      expect(portOf(result)).toBe(2);
      expect(result.key).toBeDefined();
    });

    it("should copy properties from the specified index channel", () => {
      const channels = [
        aiChannel("1", 0, "ai_accel", { channel: 3 }),
        aiChannel("2", 1, "ai_bridge"),
      ];
      const result = Channel.createNextAI(channels, { copy: "1" });
      expect(result.type).toBe("ai_accel");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(portOf(result)).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should give a new voltage channel a range of ±10 V", () => {
      expect(Channel.createNextAI([])).toMatchObject({ minVal: -10, maxVal: 10 });
    });

    it("should give a copy on another device that device's range", () => {
      const source = { ...aiChannel("1", 0), device: "a", minVal: -60, maxVal: 60 };
      expect(Channel.createNextAI([source], { device: "b" })).toMatchObject({
        minVal: -10,
        maxVal: 10,
      });
    });

    it("should keep the range of a scaled voltage copy on another device", () => {
      const customScale: NI.Task.Scale = {
        type: "linear",
        slope: 2,
        yIntercept: 0,
        preScaledUnits: "Volts",
        scaledUnits: "psi",
      };
      const source = {
        ...aiChannel("1", 0),
        device: "a",
        minVal: 0,
        maxVal: 30,
        customScale,
      };
      expect(Channel.createNextAI([source], { device: "b" })).toMatchObject({
        minVal: 0,
        maxVal: 30,
      });
    });

    it("should keep the range of a current copy on another device", () => {
      const source = {
        ...aiChannel("1", 0, "ai_current"),
        device: "a",
        minVal: -0.02,
        maxVal: 0.02,
      };
      expect(Channel.createNextAI([source], { device: "b" })).toMatchObject({
        minVal: -0.02,
        maxVal: 0.02,
      });
    });

    it("should keep the range of the copied channel", () => {
      const source = { ...aiChannel("1", 0), minVal: -1, maxVal: 1 };
      expect(Channel.createNextAI([source])).toMatchObject({ minVal: -1, maxVal: 1 });
    });

    it("should copy the last channel when no channel to copy is given", () => {
      const channels = [aiChannel("1", 0, "ai_accel"), aiChannel("2", 1, "ai_bridge")];
      const result = Channel.createNextAI(channels);
      expect(result.type).toBe("ai_bridge");
      expect(portOf(result)).toBe(2);
    });

    it("should correctly increment port when duplicating with multiple existing channels", () => {
      const channels = [
        aiChannel("1", 0),
        aiChannel("2", 1),
        aiChannel("3", 2),
        aiChannel("4", 3),
      ];
      const firstDuplicate = Channel.createNextAI(channels, { copy: "1" });
      expect(portOf(firstDuplicate)).toBe(4);

      const channelsWithFirstDuplicate = [...channels, firstDuplicate];
      const secondDuplicate = Channel.createNextAI(channelsWithFirstDuplicate, {
        copy: "1",
      });
      expect(portOf(secondDuplicate)).toBe(5);

      const thirdDuplicate = Channel.createNextAI(channelsWithFirstDuplicate, {
        copy: "3",
      });
      expect(portOf(thirdDuplicate)).toBe(5);
    });

    it("should handle non-sequential ports correctly", () => {
      const channels = [aiChannel("1", 0), aiChannel("2", 2), aiChannel("3", 5)];
      const result = Channel.createNextAI(channels, { copy: "1" });
      expect(portOf(result)).toBe(1);
    });

    it("should give the built-in temperature sensor no port", () => {
      const result = Channel.createNextAI([NI.Task.createAIChannel("ai_temp_builtin")]);
      expect(portOf(result)).toBeUndefined();
    });
  });

  describe("Channel.createNextAO", () => {
    it("should create a new AO channel with port 0 when no channels exist", () => {
      const channels: NI.Task.AOChannel[] = [];
      const result = Channel.createNextAO(channels);
      expect(result.port).toBe(0);
      expect(result.key.length).toBeGreaterThan(0);
      expect(result).toMatchObject({ cmdChannel: 0, stateChannel: 0 });
    });

    it("should create a new AI channel with the next available port number", () => {
      const channels: NI.Task.AOChannel[] = [
        ni.aoVoltageChannelZ.parse({
          type: "ao_voltage",
          key: "1",
          port: 0,
          cmdChannel: 3,
          stateChannel: 10,
        }),
        ni.aoVoltageChannelZ.parse({
          type: "ao_voltage",
          key: "2",
          port: 1,
          cmdChannel: 4,
          stateChannel: 11,
        }),
      ];
      const result = Channel.createNextAO(channels);
      expect(result.port).toBe(2);
      expect(result.key).toBeDefined();
      expect(result).toMatchObject({ cmdChannel: 0, stateChannel: 0 });
    });

    it("should copy a commanded channel without its Synnax channels", () => {
      const channels: NI.Task.AOChannel[] = [
        ni.aoCurrentChannelZ.parse({
          type: "ao_current",
          key: "1",
          port: 0,
          cmdChannel: 3,
          stateChannel: 10,
          cmdChannelName: "named_cmd",
        }),
      ];
      const result = Channel.createNextAO(channels, { copy: "1" });
      expect(result).toMatchObject({
        type: "ao_current",
        port: 1,
        cmdChannel: 0,
        stateChannel: 0,
        cmdChannelName: "",
      });
    });

    it("should copy properties from the specified index channel", () => {
      const channels: NI.Task.AOChannel[] = [
        { ...NI.Task.createAOChannel("ao_func_gen"), key: "1", port: 0 },
        { ...NI.Task.createAOChannel("ao_current"), key: "2", port: 1 },
      ];
      const result = Channel.createNextAO(channels, { copy: "1" });
      expect(result.type).toBe("ao_func_gen");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result).not.toHaveProperty("cmdChannel");
    });
  });

  describe("Channel.createNextCI", () => {
    it("should create a new CI channel with port 0 when no channels exist", () => {
      const channels: NI.Task.CIChannel[] = [];
      const result = Channel.createNextCI(channels);
      expect(result.port).toBe(0);
      expect(result.key).toBeDefined();
      expect(result.channel).toBe(0);
      expect(result.type).toBe("ci_frequency");
    });

    it("should create a new CI channel with the next available port number", () => {
      const channels: NI.Task.CIChannel[] = [
        { ...NI.Task.createCIChannel(), key: "1", port: 0 },
        { ...NI.Task.createCIChannel(), key: "2", port: 1 },
      ];
      const result = Channel.createNextCI(channels);
      expect(result.port).toBe(2);
      expect(result.key).toBeDefined();
    });

    it("should copy properties from the specified index channel", () => {
      const channels: NI.Task.CIChannel[] = [
        { ...NI.Task.createCIChannel("ci_frequency"), key: "1", port: 0, channel: 3 },
        { ...NI.Task.createCIChannel("ci_frequency"), key: "2", port: 1 },
      ];
      const result = Channel.createNextCI(channels, { copy: "1" });
      expect(result.type).toBe("ci_frequency");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should copy properties from ci_edge_count channel type", () => {
      const channels: NI.Task.CIChannel[] = [
        { ...NI.Task.createCIChannel("ci_edge_count"), key: "1", port: 0, channel: 3 },
        { ...NI.Task.createCIChannel("ci_frequency"), key: "2", port: 1 },
      ];
      const result = Channel.createNextCI(channels, { copy: "1" });
      expect(result.type).toBe("ci_edge_count");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should copy properties from ci_period channel type", () => {
      const channels: NI.Task.CIChannel[] = [
        { ...NI.Task.createCIChannel("ci_period"), key: "1", port: 0, channel: 3 },
        { ...NI.Task.createCIChannel("ci_frequency"), key: "2", port: 1 },
      ];
      const result = Channel.createNextCI(channels, { copy: "1" });
      expect(result.type).toBe("ci_period");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should copy properties from ci_pulse_width channel type", () => {
      const channels: NI.Task.CIChannel[] = [
        { ...NI.Task.createCIChannel("ci_pulse_width"), key: "1", port: 0, channel: 3 },
        { ...NI.Task.createCIChannel("ci_frequency"), key: "2", port: 1 },
      ];
      const result = Channel.createNextCI(channels, { copy: "1" });
      expect(result.type).toBe("ci_pulse_width");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should copy properties from ci_semi_period channel type", () => {
      const channels: NI.Task.CIChannel[] = [
        { ...NI.Task.createCIChannel("ci_semi_period"), key: "1", port: 0, channel: 3 },
        { ...NI.Task.createCIChannel("ci_frequency"), key: "2", port: 1 },
      ];
      const result = Channel.createNextCI(channels, { copy: "1" });
      expect(result.type).toBe("ci_semi_period");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should copy properties from ci_two_edge_sep channel type", () => {
      const channels: NI.Task.CIChannel[] = [
        {
          ...NI.Task.createCIChannel("ci_two_edge_sep"),
          key: "1",
          port: 0,
          channel: 3,
        },
        { ...NI.Task.createCIChannel("ci_frequency"), key: "2", port: 1 },
      ];
      const result = Channel.createNextCI(channels, { copy: "1" });
      expect(result.type).toBe("ci_two_edge_sep");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });
  });
});
