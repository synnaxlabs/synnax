// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Task } from "@/hardware/ni/task";
import {
  createAIChannel,
  createAOChannel,
  createCIChannel,
  createDIChannel,
  createDOChannel,
} from "@/hardware/ni/task/createChannel";

describe("createChannel", () => {
  describe("createDIChannel", () => {
    it("should create a new DI channel with line 0 when no channels exist", () => {
      const channels: Task.DIChannel[] = [];
      const result = createDIChannel(channels);
      expect(result).toEqual({
        channel: 0,
        key: expect.any(String),
        line: 0,
        port: 0,
        name: "",
        type: "digital_input",
        enabled: true,
      });
    });
    it("should create a new DI channel with the next available line number", () => {
      const channels: Task.DIChannel[] = [
        { ...Task.ZERO_DI_CHANNEL, key: "1", line: 0, channel: 3 },
        { ...Task.ZERO_DI_CHANNEL, key: "2", line: 1, channel: 4 },
      ];
      const result = createDIChannel(channels);
      expect(result.line).toBe(2);
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.channel).toBe(0);
    });
  });

  describe("createDOChannel", () => {
    it("should create a new DO channel with line 0 when no channels exist", () => {
      const channels: Task.DOChannel[] = [];
      const result = createDOChannel(channels);
      expect(result).toEqual({
        cmdChannel: 0,
        stateChannel: 0,
        key: expect.any(String),
        line: 0,
        port: 0,
        cmdChannelName: "",
        stateChannelName: "",
        type: "digital_output",
        enabled: true,
      });
    });

    it("should create a new DO channel with the next available line number", () => {
      const channels: Task.DOChannel[] = [
        { ...Task.ZERO_DO_CHANNEL, key: "1", line: 0 },
        { ...Task.ZERO_DO_CHANNEL, key: "2", line: 1 },
      ];
      const result = createDOChannel(channels);
      expect(result.line).toBe(2);
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
    });
  });

  describe("createAIChannel", () => {
    it("should create a new AI channel with port 0 when no channels exist", () => {
      const channels: Task.AIChannel[] = [];
      const result = createAIChannel(channels);
      expect(result.port).toBe(0);
      expect(result.key).toBeDefined();
      expect(result.channel).toBe(0);
    });

    it("should create a new AI channel with the next available port number", () => {
      const channels: Task.AIChannel[] = [
        { ...Task.ZERO_AI_CHANNEL, key: "1", port: 0 },
        { ...Task.ZERO_AI_CHANNEL, key: "2", port: 1 },
      ];
      const result = createAIChannel(channels);
      expect(result.port).toBe(2);
      expect(result.key).toBeDefined();
    });

    it("should copy properties from the specified index channel", () => {
      const channels: Task.AIChannel[] = [
        { ...Task.ZERO_AI_CHANNELS.ai_accel, key: "1", port: 0, channel: 3 },
        { ...Task.ZERO_AI_CHANNELS.ai_bridge, key: "2", port: 1 },
      ];
      const result = createAIChannel(channels, "1");
      expect(result.type).toBe("ai_accel");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });
  });

  describe("createAOChannel", () => {
    it("should create a new A) channel with port 0 when no channels exist", () => {
      const channels: Task.AOChannel[] = [];
      const result = createAOChannel(channels);
      expect(result.port).toBe(0);
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.cmdChannel).toBe(0);
      expect(result.stateChannel).toBe(0);
    });

    it("should create a new AI channel with the next available port number", () => {
      const channels: Task.AOChannel[] = [
        { ...Task.ZERO_AO_CHANNEL, key: "1", port: 0, cmdChannel: 3, stateChannel: 10 },
        { ...Task.ZERO_AO_CHANNEL, key: "2", port: 1, cmdChannel: 4, stateChannel: 11 },
      ];
      const result = createAOChannel(channels);
      expect(result.port).toBe(2);
      expect(result.key).toBeDefined();
      expect(result.cmdChannel).toBe(0);
      expect(result.stateChannel).toBe(0);
    });

    it("should copy properties from the specified index channel", () => {
      const channels: Task.AOChannel[] = [
        {
          ...Task.ZERO_AO_CHANNELS.ao_func_gen,
          key: "1",
          port: 0,
          cmdChannel: 3,
          stateChannel: 10,
        },
        {
          ...Task.ZERO_AO_CHANNELS.ao_current,
          key: "2",
          port: 1,
          cmdChannel: 4,
          stateChannel: 11,
        },
      ];
      const result = createAOChannel(channels, "1");
      expect(result.type).toBe("ao_func_gen");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.cmdChannel).toBe(0);
      expect(result.stateChannel).toBe(0);
    });
  });

  describe("createCIChannel", () => {
    it("should create a new CI channel with port 0 when no channels exist", () => {
      const channels: Task.CIChannel[] = [];
      const result = createCIChannel(channels);
      expect(result.port).toBe(0);
      expect(result.key).toBeDefined();
      expect(result.channel).toBe(0);
      expect(result.type).toBe("ci_frequency");
    });

    it("should create a new CI channel with the next available port number", () => {
      const channels: Task.CIChannel[] = [
        { ...Task.ZERO_CI_CHANNEL, key: "1", port: 0 },
        { ...Task.ZERO_CI_CHANNEL, key: "2", port: 1 },
      ];
      const result = createCIChannel(channels);
      expect(result.port).toBe(2);
      expect(result.key).toBeDefined();
    });

    it("should copy properties from the specified index channel", () => {
      const channels: Task.CIChannel[] = [
        { ...Task.ZERO_CI_CHANNELS.ci_frequency, key: "1", port: 0, channel: 3 },
        { ...Task.ZERO_CI_CHANNELS.ci_frequency, key: "2", port: 1 },
      ];
      const result = createCIChannel(channels, "1");
      expect(result.type).toBe("ci_frequency");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should copy properties from ci_edge_count channel type", () => {
      const channels: Task.CIChannel[] = [
        { ...Task.ZERO_CI_CHANNELS.ci_edge_count, key: "1", port: 0, channel: 3 },
        { ...Task.ZERO_CI_CHANNELS.ci_frequency, key: "2", port: 1 },
      ];
      const result = createCIChannel(channels, "1");
      expect(result.type).toBe("ci_edge_count");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should copy properties from ci_period channel type", () => {
      const channels: Task.CIChannel[] = [
        { ...Task.ZERO_CI_CHANNELS.ci_period, key: "1", port: 0, channel: 3 },
        { ...Task.ZERO_CI_CHANNELS.ci_frequency, key: "2", port: 1 },
      ];
      const result = createCIChannel(channels, "1");
      expect(result.type).toBe("ci_period");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should copy properties from ci_pulse_width channel type", () => {
      const channels: Task.CIChannel[] = [
        { ...Task.ZERO_CI_CHANNELS.ci_pulse_width, key: "1", port: 0, channel: 3 },
        { ...Task.ZERO_CI_CHANNELS.ci_frequency, key: "2", port: 1 },
      ];
      const result = createCIChannel(channels, "1");
      expect(result.type).toBe("ci_pulse_width");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should copy properties from ci_semi_period channel type", () => {
      const channels: Task.CIChannel[] = [
        { ...Task.ZERO_CI_CHANNELS.ci_semi_period, key: "1", port: 0, channel: 3 },
        { ...Task.ZERO_CI_CHANNELS.ci_frequency, key: "2", port: 1 },
      ];
      const result = createCIChannel(channels, "1");
      expect(result.type).toBe("ci_semi_period");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should copy properties from ci_two_edge_sep channel type", () => {
      const channels: Task.CIChannel[] = [
        { ...Task.ZERO_CI_CHANNELS.ci_two_edge_sep, key: "1", port: 0, channel: 3 },
        { ...Task.ZERO_CI_CHANNELS.ci_frequency, key: "2", port: 1 },
      ];
      const result = createCIChannel(channels, "1");
      expect(result.type).toBe("ci_two_edge_sep");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });
  });
});
