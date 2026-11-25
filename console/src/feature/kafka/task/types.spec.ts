// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Kafka } from "@/feature/kafka";

describe("Kafka Task Types", () => {
  describe("deployReadConfigZ", () => {
    const ICD = "6f1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d";
    const createRecord = (recordKey: string, disabled = false) => ({
      key: recordKey === "" ? "any" : recordKey,
      recordKey,
      disabled,
      icd: ICD,
      messages: [],
    });
    const createTopic = (name: string, records = [createRecord("")]) => ({
      key: name === "" ? "empty" : name,
      name,
      records,
    });
    const valid = { device: "dev", topics: [createTopic("t")] };

    it("should accept a complete config", () => {
      expect(Kafka.Task.deployReadConfigZ.safeParse(valid).success).toBe(true);
    });

    it("should reject an empty topic", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        topics: [createTopic("")],
      });
      expect(result.error?.issues).toEqual([
        expect.objectContaining({
          message: "Topic is required",
          path: ["topics", 0, "name"],
        }),
      ]);
    });

    it("should reject two topics with one name", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        topics: [createTopic("t"), { ...createTopic("t"), key: "b" }],
      });
      expect(result.error?.issues).toEqual([
        expect.objectContaining({
          message: 'Topic "t" is listed more than once',
          path: ["topics", 1, "name"],
        }),
      ]);
    });

    it("should accept a topic with no records beside one that reads", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        topics: [createTopic("t"), createTopic("u", [])],
      });
      expect(result.success).toBe(true);
    });

    it("should reject a config with every record disabled", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        topics: [createTopic("t", [createRecord("", true)]), createTopic("u", [])],
      });
      expect(result.error?.issues[0].message).toBe(
        "At least one record must be enabled",
      );
    });

    it.each([
      ["", "Only one record can read any key"],
      ["pump", 'Record key "pump" is used by another record'],
    ])("should reject two enabled records with the key %j", (recordKey, message) => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        topics: [
          createTopic("t", [
            createRecord(recordKey),
            { ...createRecord(recordKey), key: "b" },
          ]),
        ],
      });
      expect(result.error?.issues).toEqual([
        expect.objectContaining({
          message,
          path: ["topics", 0, "records", 1, "recordKey"],
        }),
      ]);
    });

    it("should accept one record key in two topics", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        topics: [
          createTopic("t", [createRecord("pump")]),
          createTopic("u", [{ ...createRecord("pump"), key: "b" }]),
        ],
      });
      expect(result.success).toBe(true);
    });

    it("should accept a disabled record that shares a key with an enabled one", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        topics: [
          createTopic("t", [
            createRecord("pump"),
            { ...createRecord("pump", true), key: "b" },
          ]),
        ],
      });
      expect(result.success).toBe(true);
    });
  });

  describe("deployWriteConfigZ", () => {
    const valid = { device: "dev", topic: "t", channels: [{ key: "c1", channel: 1 }] };

    it("should accept a complete config and fill the record shape", () => {
      const result = Kafka.Task.deployWriteConfigZ.parse(valid);
      expect(result.record.valuePointer).toBe("/value");
      expect(result.record.timeFormat).toBe("unix_ns");
      expect(result.recordKey).toBe("channel_name");
    });

    it("should reject a channel that is not selected", () => {
      const result = Kafka.Task.deployWriteConfigZ.safeParse({
        ...valid,
        channels: [{ key: "c1", channel: 0 }],
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toBe("Channel is required");
    });

    it("should accept a disabled channel that is not selected", () => {
      const result = Kafka.Task.deployWriteConfigZ.safeParse({
        ...valid,
        channels: [...valid.channels, { key: "c2", channel: 0, disabled: true }],
      });
      expect(result.success).toBe(true);
    });

    it("should reject an empty value pointer", () => {
      const result = Kafka.Task.deployWriteConfigZ.safeParse({
        ...valid,
        record: { valuePointer: "" },
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toBe("Value pointer is required");
    });
  });

  describe("deployICDWriteConfigZ", () => {
    const createTarget = (topic: string, disabled = false) => ({
      key: topic === "" ? "empty" : topic,
      topic,
      disabled,
      icd: "6f1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d",
      message: { message: "8d6a1ec4-7c8e-4f0a-9b8e-0c7f0d5c1a02", bindings: [] },
    });
    const issuesOf = (targets: ReturnType<typeof createTarget>[]) =>
      (
        Kafka.Task.deployICDWriteConfigZ.safeParse({ device: "dev", targets }).error
          ?.issues ?? []
      ).map(({ message, path }) => ({ message, path: path.join(".") }));

    it("should accept a target with a topic", () => {
      expect(issuesOf([createTarget("plant.pumps")])).toEqual([]);
    });

    it("should reject an enabled target with no topic", () => {
      expect(issuesOf([createTarget("plant.pumps"), createTarget("")])).toEqual([
        { message: "Topic is required", path: "targets.1.topic" },
      ]);
    });

    it("should reject a config whose targets are all disabled", () => {
      expect(issuesOf([createTarget("plant.pumps", true)])).toEqual([
        { message: "At least one target must be enabled", path: "targets" },
      ]);
    });

    it("should accept a disabled target with no topic", () => {
      expect(issuesOf([createTarget("plant.pumps"), createTarget("", true)])).toEqual(
        [],
      );
    });
  });

  describe("isString", () => {
    it.each(["string", "json", "uuid"])("should send %s as a string", (dt) => {
      expect(Kafka.Task.isString(dt)).toBe(true);
    });

    it.each(["float64", "uint8", "timestamp"])(
      "should not send %s as a string",
      (dt) => {
        expect(Kafka.Task.isString(dt)).toBe(false);
      },
    );
  });
});
