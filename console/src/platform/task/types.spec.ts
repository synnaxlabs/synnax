// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";

describe("Hardware Task Types", () => {
  describe("Channel Validation", () => {
    // Create schemas that use the validation functions, adapting them to work with refine
    const channelsArrayZ = z.array(Task.channelZ).check(Task.validateChannels);
    const readChannelsArrayZ = z
      .array(Task.readChannelZ)
      .check(Task.validateReadChannels);
    const writeChannelsArrayZ = z
      .array(Task.writeChannelZ)
      .check(Task.validateWriteChannels);

    describe("validateChannels", () => {
      it("should report errors for duplicate channel keys", () => {
        const channels: Task.Channel[] = [
          { ...Task.ZERO_CHANNEL, enabled: true, key: "duplicate" },
          { ...Task.ZERO_CHANNEL, enabled: true, key: "duplicate" },
        ];

        const result = channelsArrayZ.safeParse(channels);

        expect(result.success).toBe(false);
        if (!result.success) {
          expect(result.error.issues.length).toBe(2); // Two issues - one for each duplicate key
          expect(
            result.error.issues.some(
              (issue) =>
                issue.message === "Key duplicate is used for multiple channels",
            ),
          ).toBe(true);
        }
      });

      it("should not report errors for valid channels", () => {
        const channels: Task.Channel[] = [
          { ...Task.ZERO_CHANNEL, enabled: true, key: "channel1" },
          { ...Task.ZERO_CHANNEL, enabled: true, key: "channel2" },
        ];

        const result = channelsArrayZ.safeParse(channels);

        expect(result.success).toBe(true);
      });
    });

    describe("validateReadChannels", () => {
      it("should report errors for duplicate channel values", () => {
        const channels: Task.ReadChannel[] = [
          { ...Task.ZERO_READ_CHANNEL, enabled: true, key: "channel1", channel: 1 },
          { ...Task.ZERO_READ_CHANNEL, enabled: true, key: "channel2", channel: 1 },
        ];

        const result = readChannelsArrayZ.safeParse(channels);

        expect(result.success).toBe(false);
        if (!result.success) {
          expect(result.error.issues.length).toBe(2); // Two issues - one for each channel with duplicate value
          expect(
            result.error.issues.some(
              (issue) =>
                issue.message ===
                "Synnax channel with key 1 is used for multiple channels",
            ),
          ).toBe(true);
        }
      });

      it("should not report errors for valid read channels", () => {
        const channels: Task.ReadChannel[] = [
          { ...Task.ZERO_READ_CHANNEL, enabled: true, key: "channel1", channel: 1 },
          { ...Task.ZERO_READ_CHANNEL, enabled: true, key: "channel2", channel: 2 },
        ];

        const result = readChannelsArrayZ.safeParse(channels);

        expect(result.success).toBe(true);
      });

      it("should ignore channel value 0", () => {
        const channels: Task.ReadChannel[] = [
          { ...Task.ZERO_READ_CHANNEL, enabled: true, key: "channel1", channel: 0 },
          { ...Task.ZERO_READ_CHANNEL, enabled: true, key: "channel2", channel: 0 },
        ];

        const result = readChannelsArrayZ.safeParse(channels);

        expect(result.success).toBe(true);
      });

      it("should report errors for duplicate channel names", () => {
        const channels: Task.ReadChannel[] = [
          {
            ...Task.ZERO_READ_CHANNEL,
            enabled: true,
            key: "channel1",
            channel: 1,
            name: "temp",
          },
          {
            ...Task.ZERO_READ_CHANNEL,
            enabled: true,
            key: "channel2",
            channel: 2,
            name: "temp",
          },
        ];

        const result = readChannelsArrayZ.safeParse(channels);

        expect(result.success).toBe(false);
        if (!result.success)
          expect(
            result.error.issues.some(
              (issue) => issue.message === "Name temp is used for multiple channels",
            ),
          ).toBe(true);
      });

      it("should ignore empty channel names", () => {
        const channels: Task.ReadChannel[] = [
          {
            ...Task.ZERO_READ_CHANNEL,
            enabled: true,
            key: "channel1",
            channel: 1,
            name: "",
          },
          {
            ...Task.ZERO_READ_CHANNEL,
            enabled: true,
            key: "channel2",
            channel: 1,
            name: "",
          },
        ];

        const result = readChannelsArrayZ.safeParse(channels);

        expect(result.success).toBe(false);
        if (!result.success) {
          expect(result.error.issues.length).toBe(2);
          expect(
            result.error.issues.every((issue) => !issue.message.startsWith("Name")),
          ).toBe(true);
        }
      });
    });

    describe("validateWriteChannels", () => {
      it("should report errors for duplicate cmd channels", () => {
        const channels: Task.WriteChannel[] = [
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel1",
            cmdChannel: 1,
            stateChannel: 2,
          },
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel2",
            cmdChannel: 1,
            stateChannel: 3,
          },
        ];

        const result = writeChannelsArrayZ.safeParse(channels);

        expect(result.success).toBe(false);
        if (!result.success) {
          expect(result.error.issues.length).toBe(2);
          expect(
            result.error.issues.some(
              (issue) =>
                issue.message ===
                "Synnax channel with key 1 is used on multiple channels",
            ),
          ).toBe(true);
        }
      });

      it("should report errors for duplicate state channels", () => {
        const channels: Task.WriteChannel[] = [
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel1",
            cmdChannel: 1,
            stateChannel: 3,
          },
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel2",
            cmdChannel: 2,
            stateChannel: 3,
          },
        ];

        const result = writeChannelsArrayZ.safeParse(channels);

        expect(result.success).toBe(false);
        if (!result.success) {
          expect(result.error.issues.length).toBe(2);
          expect(
            result.error.issues.some(
              (issue) =>
                issue.message ===
                "Synnax channel with key 3 is used for multiple channels",
            ),
          ).toBe(true);
        }
      });

      it("should not report errors for valid write channels", () => {
        const channels: Task.WriteChannel[] = [
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel1",
            cmdChannel: 1,
            stateChannel: 2,
          },
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel2",
            cmdChannel: 3,
            stateChannel: 4,
          },
        ];

        const result = writeChannelsArrayZ.safeParse(channels);

        expect(result.success).toBe(true);
      });

      it("should ignore channel values of 0", () => {
        const channels: Task.WriteChannel[] = [
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel1",
            cmdChannel: 1,
            stateChannel: 0,
          },
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel2",
            cmdChannel: 0,
            stateChannel: 0,
          },
        ];

        const result = writeChannelsArrayZ.safeParse(channels);

        expect(result.success).toBe(true);
      });

      it("should report errors for duplicate cmd channel names", () => {
        const channels: Task.WriteChannel[] = [
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel1",
            cmdChannel: 1,
            stateChannel: 2,
            cmdChannelName: "cmd",
          },
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel2",
            cmdChannel: 3,
            stateChannel: 4,
            cmdChannelName: "cmd",
          },
        ];

        const result = writeChannelsArrayZ.safeParse(channels);

        expect(result.success).toBe(false);
        if (!result.success)
          expect(
            result.error.issues.some(
              (issue) => issue.message === "Name cmd is used for multiple channels",
            ),
          ).toBe(true);
      });

      it("should report errors for duplicate state channel names", () => {
        const channels: Task.WriteChannel[] = [
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel1",
            cmdChannel: 1,
            stateChannel: 2,
            stateChannelName: "state",
          },
          {
            ...Task.ZERO_WRITE_CHANNEL,
            enabled: true,
            key: "channel2",
            cmdChannel: 3,
            stateChannel: 4,
            stateChannelName: "state",
          },
        ];

        const result = writeChannelsArrayZ.safeParse(channels);

        expect(result.success).toBe(false);
        if (!result.success)
          expect(
            result.error.issues.some(
              (issue) => issue.message === "Name state is used for multiple channels",
            ),
          ).toBe(true);
      });
    });
  });

  describe("Stream Rate Validation", () => {
    // Create a schema with a custom refine function that uses validateStreamRate
    const streamRateConfigZ = z
      .object({
        sampleRate: z.number(),
        streamRate: z.number(),
      })
      .check(Task.validateStreamRate);

    it("should return success if sample rate >= stream rate", () => {
      const validResult = streamRateConfigZ.safeParse({
        sampleRate: 100,
        streamRate: 50,
      });
      expect(validResult.success).toBe(true);

      const alsoValidResult = streamRateConfigZ.safeParse({
        sampleRate: 100,
        streamRate: 100,
      });
      expect(alsoValidResult.success).toBe(true);
    });

    it("should return error if sample rate < stream rate", () => {
      const invalidResult = streamRateConfigZ.safeParse({
        sampleRate: 50,
        streamRate: 100,
      });
      expect(invalidResult.success).toBe(false);
    });

    it("should provide the correct error message", () => {
      const invalidResult = streamRateConfigZ.safeParse({
        sampleRate: 0,
        streamRate: 1,
      });
      expect(invalidResult.success).toBe(false);
      if (!invalidResult.success) {
        expect(invalidResult.error.issues[0].path).toEqual(["streamRate"]);
        expect(invalidResult.error.issues[0].message).toBe(
          "Stream rate must be less than or equal to the sample rate",
        );
      }
    });
  });
});
