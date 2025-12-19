// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { describe, expect, it, vi } from "vitest";

import {
  collectDataChannelsForValidation,
  collectIndexChannelForValidation,
  collectWriteChannelsForValidation,
  getChannelNameToCreate,
  shouldCreateChannel,
  shouldCreateChannels,
  validateChannelNames,
} from "@/hardware/common/task/validateChannelNames";

const createMockClient = () => ({
  channels: {
    retrieve: vi.fn(),
  },
});

describe("validateChannelNames", () => {
  describe("shouldCreateChannel", () => {
    it("should return true for zero key", async () => {
      const client = createMockClient();
      expect(await shouldCreateChannel(client as any, 0)).toBe(true);
      expect(client.channels.retrieve).not.toHaveBeenCalled();
    });

    it("should return true for null key", async () => {
      const client = createMockClient();
      expect(await shouldCreateChannel(client as any, null)).toBe(true);
      expect(client.channels.retrieve).not.toHaveBeenCalled();
    });

    it("should return true for undefined key", async () => {
      const client = createMockClient();
      expect(await shouldCreateChannel(client as any, undefined)).toBe(true);
      expect(client.channels.retrieve).not.toHaveBeenCalled();
    });

    it("should return false when channel exists", async () => {
      const client = createMockClient();
      client.channels.retrieve.mockResolvedValue({ key: 123, name: "test" });
      expect(await shouldCreateChannel(client as any, 123)).toBe(false);
      expect(client.channels.retrieve).toHaveBeenCalledWith(123);
    });

    it("should return true when channel was deleted (NotFoundError)", async () => {
      const client = createMockClient();
      client.channels.retrieve.mockRejectedValue(
        new NotFoundError("channel not found"),
      );
      expect(await shouldCreateChannel(client as any, 123)).toBe(true);
      expect(client.channels.retrieve).toHaveBeenCalledWith(123);
    });

    it("should rethrow non-NotFoundError errors", async () => {
      const client = createMockClient();
      const error = new Error("network error");
      client.channels.retrieve.mockRejectedValue(error);
      await expect(shouldCreateChannel(client as any, 123)).rejects.toThrow(
        "network error",
      );
    });
  });

  describe("shouldCreateChannels", () => {
    it("should check multiple keys in parallel", async () => {
      const client = createMockClient();
      client.channels.retrieve
        .mockResolvedValueOnce({ key: 1 })
        .mockRejectedValueOnce(new NotFoundError("not found"))
        .mockResolvedValueOnce({ key: 3 });

      const result = await shouldCreateChannels(client as any, [1, 2, 3]);

      expect(result).toEqual([false, true, false]);
      expect(client.channels.retrieve).toHaveBeenCalledTimes(3);
    });

    it("should handle zero/null/undefined keys without calling retrieve", async () => {
      const client = createMockClient();
      client.channels.retrieve.mockResolvedValue({ key: 5 });

      const result = await shouldCreateChannels(client as any, [0, null, undefined, 5]);

      expect(result).toEqual([true, true, true, false]);
      expect(client.channels.retrieve).toHaveBeenCalledTimes(1);
      expect(client.channels.retrieve).toHaveBeenCalledWith(5);
    });

    it("should return empty array for empty input", async () => {
      const client = createMockClient();
      const result = await shouldCreateChannels(client as any, []);
      expect(result).toEqual([]);
      expect(client.channels.retrieve).not.toHaveBeenCalled();
    });
  });

  describe("getChannelNameToCreate", () => {
    it("should return prename when provided", () => {
      expect(getChannelNameToCreate("custom_name", "default_name")).toBe("custom_name");
    });

    it("should return default name when prename is null", () => {
      expect(getChannelNameToCreate(null, "default_name")).toBe("default_name");
    });

    it("should return default name when prename is undefined", () => {
      expect(getChannelNameToCreate(undefined, "default_name")).toBe("default_name");
    });

    it("should return default name when prename is empty string", () => {
      expect(getChannelNameToCreate("", "default_name")).toBe("default_name");
    });
  });

  describe("validateChannelNames", () => {
    it("should not throw for empty array", async () => {
      const client = createMockClient();
      await expect(validateChannelNames(client as any, [])).resolves.not.toThrow();
      expect(client.channels.retrieve).not.toHaveBeenCalled();
    });

    it("should not throw when no channels exist", async () => {
      const client = createMockClient();
      client.channels.retrieve.mockRejectedValue(new NotFoundError("not found"));

      await expect(
        validateChannelNames(client as any, ["channel_a", "channel_b"]),
      ).resolves.not.toThrow();
    });

    it("should throw when a single channel already exists", async () => {
      const client = createMockClient();
      client.channels.retrieve.mockResolvedValue({ name: "existing_channel" });

      await expect(
        validateChannelNames(client as any, ["existing_channel"]),
      ).rejects.toThrow(
        "Cannot configure task: the following channel already exists: existing_channel",
      );
    });

    it("should throw when multiple channels already exist", async () => {
      const client = createMockClient();
      client.channels.retrieve.mockResolvedValue({ name: "test" });

      await expect(
        validateChannelNames(client as any, ["channel_a", "channel_b"]),
      ).rejects.toThrow(
        "Cannot configure task: the following channels already exist: channel_a, channel_b",
      );
    });

    it("should only report existing channels", async () => {
      const client = createMockClient();
      client.channels.retrieve
        .mockResolvedValueOnce({ name: "existing" })
        .mockRejectedValueOnce(new NotFoundError("not found"))
        .mockResolvedValueOnce({ name: "also_existing" });

      await expect(
        validateChannelNames(client as any, [
          "existing",
          "new_channel",
          "also_existing",
        ]),
      ).rejects.toThrow(
        "Cannot configure task: the following channels already exist: existing, also_existing",
      );
    });

    it("should rethrow non-NotFoundError errors", async () => {
      const client = createMockClient();
      client.channels.retrieve.mockRejectedValue(new Error("network error"));

      await expect(validateChannelNames(client as any, ["channel_a"])).rejects.toThrow(
        "network error",
      );
    });
  });

  describe("collectIndexChannelForValidation", () => {
    it("should return shouldCreate=true and name when index doesn't exist", async () => {
      const client = createMockClient();

      const result = await collectIndexChannelForValidation(
        client as any,
        0,
        "device_time",
      );

      expect(result).toEqual({
        shouldCreate: true,
        nameToValidate: "device_time",
      });
    });

    it("should return shouldCreate=false and null name when index exists", async () => {
      const client = createMockClient();
      client.channels.retrieve.mockResolvedValue({ key: 123 });

      const result = await collectIndexChannelForValidation(
        client as any,
        123,
        "device_time",
      );

      expect(result).toEqual({
        shouldCreate: false,
        nameToValidate: null,
      });
    });

    it("should return shouldCreate=true when channel was deleted", async () => {
      const client = createMockClient();
      client.channels.retrieve.mockRejectedValue(new NotFoundError("not found"));

      const result = await collectIndexChannelForValidation(
        client as any,
        123,
        "device_time",
      );

      expect(result).toEqual({
        shouldCreate: true,
        nameToValidate: "device_time",
      });
    });
  });

  describe("collectDataChannelsForValidation", () => {
    interface TestChannel {
      id: string;
      name?: string;
      existingKey: number;
    }

    it("should collect channels that need creation", async () => {
      const client = createMockClient();
      client.channels.retrieve
        .mockResolvedValueOnce({ key: 1 })
        .mockRejectedValueOnce(new NotFoundError("not found"));

      const channels: TestChannel[] = [
        { id: "a", existingKey: 1 },
        { id: "b", existingKey: 2 },
      ];

      const result = await collectDataChannelsForValidation(
        client as any,
        channels,
        (ch) => ch.existingKey,
        (ch) => ({ prename: ch.name, defaultName: `channel_${ch.id}` }),
      );

      expect(result.toCreate).toEqual([{ id: "b", existingKey: 2 }]);
      expect(result.namesToValidate).toEqual(["channel_b"]);
    });

    it("should use prename when provided", async () => {
      const client = createMockClient();

      const channels: TestChannel[] = [
        { id: "a", name: "custom_name", existingKey: 0 },
      ];

      const result = await collectDataChannelsForValidation(
        client as any,
        channels,
        (ch) => ch.existingKey,
        (ch) => ({ prename: ch.name, defaultName: `channel_${ch.id}` }),
      );

      expect(result.namesToValidate).toEqual(["custom_name"]);
    });

    it("should check all channels in parallel", async () => {
      const client = createMockClient();
      client.channels.retrieve.mockRejectedValue(new NotFoundError("not found"));

      const channels: TestChannel[] = [
        { id: "a", existingKey: 1 },
        { id: "b", existingKey: 2 },
        { id: "c", existingKey: 3 },
      ];

      const result = await collectDataChannelsForValidation(
        client as any,
        channels,
        (ch) => ch.existingKey,
        (ch) => ({ defaultName: `channel_${ch.id}` }),
      );

      expect(result.toCreate).toHaveLength(3);
      expect(result.namesToValidate).toEqual(["channel_a", "channel_b", "channel_c"]);
      expect(client.channels.retrieve).toHaveBeenCalledTimes(3);
    });

    it("should return empty arrays for empty input", async () => {
      const client = createMockClient();

      const result = await collectDataChannelsForValidation(
        client as any,
        [],
        (ch: TestChannel) => ch.existingKey,
        (ch: TestChannel) => ({ defaultName: `channel_${ch.id}` }),
      );

      expect(result.toCreate).toEqual([]);
      expect(result.namesToValidate).toEqual([]);
    });
  });

  describe("collectWriteChannelsForValidation", () => {
    interface TestWriteChannel {
      port: number;
      cmdName?: string;
      stateName?: string;
    }

    const getChannelNames = (ch: TestWriteChannel) => ({
      cmdPrename: ch.cmdName,
      cmdDefault: `port_${ch.port}_cmd`,
      cmdIndexDefault: `port_${ch.port}_cmd_time`,
      statePrename: ch.stateName,
      stateDefault: `port_${ch.port}_state`,
    });

    it("should add to both lists when pair is null", async () => {
      const client = createMockClient();

      const channels: TestWriteChannel[] = [{ port: 1 }];

      const result = await collectWriteChannelsForValidation(
        client as any,
        channels,
        () => null,
        getChannelNames,
      );

      expect(result.commandsToCreate).toEqual([{ port: 1 }]);
      expect(result.statesToCreate).toEqual([{ port: 1 }]);
      expect(result.namesToValidate).toEqual([
        "port_1_state",
        "port_1_cmd_time",
        "port_1_cmd",
      ]);
    });

    it("should only add state when command exists", async () => {
      const client = createMockClient();
      client.channels.retrieve.mockResolvedValueOnce({ key: 100 });

      const channels: TestWriteChannel[] = [{ port: 1 }];

      const result = await collectWriteChannelsForValidation(
        client as any,
        channels,
        () => ({ command: 100, state: 0 }),
        getChannelNames,
      );

      expect(result.commandsToCreate).toEqual([]);
      expect(result.statesToCreate).toEqual([{ port: 1 }]);
      expect(result.namesToValidate).toEqual(["port_1_state"]);
    });

    it("should only add command when state exists", async () => {
      const client = createMockClient();
      client.channels.retrieve
        .mockResolvedValueOnce({ key: 200 })
        .mockRejectedValueOnce(new NotFoundError("not found"));

      const channels: TestWriteChannel[] = [{ port: 1 }];

      const result = await collectWriteChannelsForValidation(
        client as any,
        channels,
        () => ({ command: 0, state: 200 }),
        getChannelNames,
      );

      expect(result.commandsToCreate).toEqual([{ port: 1 }]);
      expect(result.statesToCreate).toEqual([]);
      expect(result.namesToValidate).toEqual(["port_1_cmd_time", "port_1_cmd"]);
    });

    it("should use prenames when provided", async () => {
      const client = createMockClient();

      const channels: TestWriteChannel[] = [
        { port: 1, cmdName: "my_cmd", stateName: "my_state" },
      ];

      const result = await collectWriteChannelsForValidation(
        client as any,
        channels,
        () => null,
        getChannelNames,
      );

      expect(result.namesToValidate).toEqual(["my_state", "my_cmd_time", "my_cmd"]);
    });

    it("should use default names when prenames are empty strings", async () => {
      const client = createMockClient();

      const channels: TestWriteChannel[] = [{ port: 1, cmdName: "", stateName: "" }];

      const result = await collectWriteChannelsForValidation(
        client as any,
        channels,
        () => null,
        getChannelNames,
      );

      expect(result.namesToValidate).toEqual([
        "port_1_state",
        "port_1_cmd_time",
        "port_1_cmd",
      ]);
    });

    it("should handle multiple channels with mixed states", async () => {
      const client = createMockClient();
      client.channels.retrieve
        .mockResolvedValueOnce({ key: 200 })
        .mockResolvedValueOnce({ key: 300 })
        .mockResolvedValueOnce({ key: 301 });

      const channels: TestWriteChannel[] = [{ port: 1 }, { port: 2 }, { port: 3 }];

      const result = await collectWriteChannelsForValidation(
        client as any,
        channels,
        (ch) => {
          if (ch.port === 1) return null;
          if (ch.port === 2) return { command: 0, state: 200 };
          return { command: 301, state: 300 };
        },
        getChannelNames,
      );

      expect(result.commandsToCreate).toEqual([{ port: 1 }, { port: 2 }]);
      expect(result.statesToCreate).toEqual([{ port: 1 }]);
      expect(result.namesToValidate).toEqual([
        "port_1_state",
        "port_1_cmd_time",
        "port_1_cmd",
        "port_2_cmd_time",
        "port_2_cmd",
      ]);
    });

    it("should return empty arrays for empty input", async () => {
      const client = createMockClient();

      const result = await collectWriteChannelsForValidation(
        client as any,
        [],
        () => null,
        getChannelNames,
      );

      expect(result.commandsToCreate).toEqual([]);
      expect(result.statesToCreate).toEqual([]);
      expect(result.namesToValidate).toEqual([]);
    });
  });
});
