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
    const valid = { device: "dev", topic: "t", records: [createRecord("")] };

    it("should accept a complete config", () => {
      expect(Kafka.Task.deployReadConfigZ.safeParse(valid).success).toBe(true);
    });

    it("should reject an empty topic", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({ ...valid, topic: "" });
      expect(result.error?.issues[0].message).toBe("Topic is required");
    });

    it("should reject a config with every record disabled", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        records: [createRecord("", true)],
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
        records: [createRecord(recordKey), { ...createRecord(recordKey), key: "b" }],
      });
      expect(result.error?.issues).toEqual([
        expect.objectContaining({ message, path: ["records", 1, "recordKey"] }),
      ]);
    });

    it("should accept a disabled record that shares a key with an enabled one", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        records: [createRecord("pump"), { ...createRecord("pump", true), key: "b" }],
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
