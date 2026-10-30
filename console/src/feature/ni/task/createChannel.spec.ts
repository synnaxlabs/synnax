// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import {
  createNextAIChannel,
  createNextAOChannel,
  createNextCIChannel,
  createNextDIChannel,
  createNextDOChannel,
} from "@/feature/ni/task/createChannel";
import { createHardware } from "@/feature/ni/task/testutil";

const createDevice = (key: string): NI.Device.Device => ({
  key,
  rack: 1,
  location: "Dev1",
  make: "NI",
  model: "PCIe-6363",
  name: key,
  configured: true,
  properties: { ...NI.Device.ZERO_PROPERTIES, hardware: createHardware() },
});

// AI channels are a discriminated union and the built-in temperature sensor variant
// carries no port, so the specs reach for one through these.
const portOf = (channel: NI.Task.AIChannel): number | undefined =>
  "port" in channel ? channel.port : undefined;

const aiChannel = (
  key: string,
  port: number,
  type: NI.Task.AIChannelType = "ai_voltage",
  extra: { channel?: number; device?: string } = {},
): NI.Task.AIChannel => {
  const channel = NI.Task.createAIChannel(type);
  if (!("port" in channel)) throw new Error(`${type} carries no port`);
  channel.port = port;
  return { ...channel, key, ...extra };
};

describe("createChannel", () => {
  describe("createNextDIChannel", () => {
    it("should create a new DI channel with line 0 when no channels exist", () => {
      const channels: NI.Task.DIChannel[] = [];
      const result = createNextDIChannel(channels);
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
      const result = createNextDIChannel(channels);
      expect(result.line).toBe(2);
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.channel).toBe(0);
    });
  });

  describe("createNextDIChannel on a device with a port list", () => {
    it("should take the first free line of the next port once a port is full", () => {
      const channels = Array.from({ length: 8 }, (_, line) => ({
        ...NI.Task.createDIChannel(),
        key: String(line),
        line,
      }));
      const result = createNextDIChannel(channels, {}, createDevice("a"));
      expect([result.port, result.line]).toEqual([1, 0]);
    });

    it("should put the channel on the given line", () => {
      const result = createNextDIChannel([], { port: 1, line: 5 }, createDevice("a"));
      expect([result.port, result.line]).toEqual([1, 5]);
    });
  });

  describe("createNextDOChannel", () => {
    it("should create a new DO channel with line 0 when no channels exist", () => {
      const channels: NI.Task.DOChannel[] = [];
      const result = createNextDOChannel(channels);
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
      const result = createNextDOChannel(channels);
      expect(result.line).toBe(2);
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
    });
  });

  describe("createNextAIChannel", () => {
    it("should create a new AI channel with port 0 when no channels exist", () => {
      const channels: NI.Task.AIChannel[] = [];
      const result = createNextAIChannel(channels);
      expect(portOf(result)).toBe(0);
      expect(result.key).toBeDefined();
      expect(result.channel).toBe(0);
    });

    it("should create a new AI channel with the next available port number", () => {
      const channels = [aiChannel("1", 0), aiChannel("2", 1)];
      const result = createNextAIChannel(channels);
      expect(portOf(result)).toBe(2);
      expect(result.key).toBeDefined();
    });

    it("should copy properties from the specified index channel", () => {
      const channels = [
        aiChannel("1", 0, "ai_accel", { channel: 3 }),
        aiChannel("2", 1, "ai_bridge"),
      ];
      const result = createNextAIChannel(channels, { copy: "1" });
      expect(result.type).toBe("ai_accel");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(portOf(result)).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should correctly increment port when duplicating with multiple existing channels", () => {
      const channels = [
        aiChannel("1", 0),
        aiChannel("2", 1),
        aiChannel("3", 2),
        aiChannel("4", 3),
      ];
      const firstDuplicate = createNextAIChannel(channels, { copy: "1" });
      expect(portOf(firstDuplicate)).toBe(4);

      const channelsWithFirstDuplicate = [...channels, firstDuplicate];
      const secondDuplicate = createNextAIChannel(channelsWithFirstDuplicate, {
        copy: "1",
      });
      expect(portOf(secondDuplicate)).toBe(5);

      const thirdDuplicate = createNextAIChannel(channelsWithFirstDuplicate, {
        copy: "3",
      });
      expect(portOf(thirdDuplicate)).toBe(5);
    });

    it("should handle non-sequential ports correctly", () => {
      const channels = [aiChannel("1", 0), aiChannel("2", 2), aiChannel("3", 5)];
      const result = createNextAIChannel(channels, { copy: "1" });
      expect(portOf(result)).toBe(1);
    });

    it("should take the lowest free port on its own device, not in the task", () => {
      const channels = [
        aiChannel("1", 0, "ai_voltage", { device: "a" }),
        aiChannel("2", 1, "ai_voltage", { device: "a" }),
        aiChannel("3", 0, "ai_voltage", { device: "b" }),
      ];
      expect(portOf(createNextAIChannel(channels, { copy: "3" }))).toBe(1);
    });

    it("should skip a port that a differential channel uses as its negative input", () => {
      const dev = createDevice("a");
      const channels = Array.from({ length: 8 }, (_, port) =>
        aiChannel(String(port), port, "ai_voltage", { device: "a" }),
      );
      const result = createNextAIChannel(channels.slice(0, 1), {}, [dev]);
      expect(portOf(result)).toBe(1);
      expect(portOf(createNextAIChannel(channels, {}, [dev]))).toBe(16);
    });

    it("should take a port whose negative input is free for a channel that copies a differential one", () => {
      const dev = createDevice("a");
      const channels = [
        aiChannel("1", 0, "ai_voltage", { device: "a" }),
        aiChannel("2", 9, "ai_voltage", { device: "a" }),
      ];
      expect(portOf(createNextAIChannel(channels, { copy: "1" }, [dev]))).toBe(2);
    });

    it("should put the channel on the given device and port", () => {
      const result = createNextAIChannel([aiChannel("1", 0)], { device: "b", port: 6 });
      expect(result.device).toBe("b");
      expect(portOf(result)).toBe(6);
    });

    it("should give the built-in temperature sensor no port", () => {
      const result = createNextAIChannel([NI.Task.createAIChannel("ai_temp_builtin")]);
      expect(portOf(result)).toBeUndefined();
    });
  });

  describe("createNextAOChannel", () => {
    it("should create a new AO channel with port 0 when no channels exist", () => {
      const channels: NI.Task.AOChannel[] = [];
      const result = createNextAOChannel(channels);
      expect(result.port).toBe(0);
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.cmdChannel).toBe(0);
      expect(result.stateChannel).toBe(0);
    });

    it("should create a new AI channel with the next available port number", () => {
      const channels: NI.Task.AOChannel[] = [
        {
          ...NI.Task.createAOChannel(),
          key: "1",
          port: 0,
          cmdChannel: 3,
          stateChannel: 10,
        },
        {
          ...NI.Task.createAOChannel(),
          key: "2",
          port: 1,
          cmdChannel: 4,
          stateChannel: 11,
        },
      ];
      const result = createNextAOChannel(channels);
      expect(result.port).toBe(2);
      expect(result.key).toBeDefined();
      expect(result.cmdChannel).toBe(0);
      expect(result.stateChannel).toBe(0);
    });

    it("should copy properties from the specified index channel", () => {
      const channels: NI.Task.AOChannel[] = [
        {
          ...NI.Task.createAOChannel("ao_func_gen"),
          key: "1",
          port: 0,
          cmdChannel: 3,
          stateChannel: 10,
        },
        {
          ...NI.Task.createAOChannel("ao_current"),
          key: "2",
          port: 1,
          cmdChannel: 4,
          stateChannel: 11,
        },
      ];
      const result = createNextAOChannel(channels, { copy: "1" });
      expect(result.type).toBe("ao_func_gen");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.cmdChannel).toBe(0);
      expect(result.stateChannel).toBe(0);
    });
  });

  describe("createNextCIChannel", () => {
    it("should create a new CI channel with port 0 when no channels exist", () => {
      const channels: NI.Task.CIChannel[] = [];
      const result = createNextCIChannel(channels);
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
      const result = createNextCIChannel(channels);
      expect(result.port).toBe(2);
      expect(result.key).toBeDefined();
    });

    it("should copy properties from the specified index channel", () => {
      const channels: NI.Task.CIChannel[] = [
        { ...NI.Task.createCIChannel("ci_frequency"), key: "1", port: 0, channel: 3 },
        { ...NI.Task.createCIChannel("ci_frequency"), key: "2", port: 1 },
      ];
      const result = createNextCIChannel(channels, { copy: "1" });
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
      const result = createNextCIChannel(channels, { copy: "1" });
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
      const result = createNextCIChannel(channels, { copy: "1" });
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
      const result = createNextCIChannel(channels, { copy: "1" });
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
      const result = createNextCIChannel(channels, { copy: "1" });
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
      const result = createNextCIChannel(channels, { copy: "1" });
      expect(result.type).toBe("ci_two_edge_sep");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should take the lowest free counter on the device of the copied channel", () => {
      const channels: NI.Task.CIChannel[] = [
        { ...NI.Task.createCIChannel("ci_frequency"), key: "1", device: "a", port: 0 },
        { ...NI.Task.createCIChannel("ci_frequency"), key: "2", device: "b", port: 0 },
        { ...NI.Task.createCIChannel("ci_frequency"), key: "3", device: "b", port: 1 },
      ];
      const result = createNextCIChannel(channels, { copy: "1" }, [createDevice("a")]);
      expect([result.device, result.port]).toEqual(["a", 1]);
    });
  });
});
