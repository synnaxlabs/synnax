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
        type: "digital_input",
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
        type: "digital_output",
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
      expect(result.port).toBe(0);
      expect(result.key).toBeDefined();
      expect(result.channel).toBe(0);
    });

    it("should create a new AI channel with the next available port number", () => {
      const channels: NI.Task.AIChannel[] = [
        { ...NI.Task.createAIChannel(), key: "1", port: 0 },
        { ...NI.Task.createAIChannel(), key: "2", port: 1 },
      ];
      const result = createNextAIChannel(channels);
      expect(result.port).toBe(2);
      expect(result.key).toBeDefined();
    });

    it("should copy properties from the specified index channel", () => {
      const channels: NI.Task.AIChannel[] = [
        { ...NI.Task.createAIChannel("ai_accel"), key: "1", port: 0, channel: 3 },
        { ...NI.Task.createAIChannel("ai_bridge"), key: "2", port: 1 },
      ];
      const result = createNextAIChannel(channels, "1");
      expect(result.type).toBe("ai_accel");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });

    it("should correctly increment port when duplicating with multiple existing channels", () => {
      const channels: NI.Task.AIChannel[] = [
        { ...NI.Task.createAIChannel(), key: "1", port: 0 },
        { ...NI.Task.createAIChannel(), key: "2", port: 1 },
        { ...NI.Task.createAIChannel(), key: "3", port: 2 },
        { ...NI.Task.createAIChannel(), key: "4", port: 3 },
      ];
      const firstDuplicate = createNextAIChannel(channels, "1");
      expect(firstDuplicate.port).toBe(4);

      const channelsWithFirstDuplicate = [...channels, firstDuplicate];
      const secondDuplicate = createNextAIChannel(channelsWithFirstDuplicate, "1");
      expect(secondDuplicate.port).toBe(5);

      const thirdDuplicate = createNextAIChannel(channelsWithFirstDuplicate, "3");
      expect(thirdDuplicate.port).toBe(5);
    });

    it("should handle non-sequential ports correctly", () => {
      const channels: NI.Task.AIChannel[] = [
        { ...NI.Task.createAIChannel(), key: "1", port: 0 },
        { ...NI.Task.createAIChannel(), key: "2", port: 2 },
        { ...NI.Task.createAIChannel(), key: "3", port: 5 },
      ];
      const result = createNextAIChannel(channels, "1");
      expect(result.port).toBe(1);
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
      const result = createNextAOChannel(channels, "1");
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
      const result = createNextCIChannel(channels, "1");
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
      const result = createNextCIChannel(channels, "1");
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
      const result = createNextCIChannel(channels, "1");
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
      const result = createNextCIChannel(channels, "1");
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
      const result = createNextCIChannel(channels, "1");
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
      const result = createNextCIChannel(channels, "1");
      expect(result.type).toBe("ci_two_edge_sep");
      expect(result.key).not.toBe("1");
      expect(result.key).not.toBe("2");
      expect(result.key.length).toBeGreaterThan(0);
      expect(result.port).toBe(2);
      expect(result.channel).not.toBe(3);
    });
  });
});
