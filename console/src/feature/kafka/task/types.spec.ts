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
  const field = {
    key: "f1",
    pointer: "/value",
    channel: 1,
    dataType: "float64",
  };

  describe("READ_SCHEMAS", () => {
    it("should validate the type literal", () => {
      expect(Kafka.Task.READ_SCHEMAS.type.parse(Kafka.Task.READ_TYPE)).toBe(
        Kafka.Task.READ_TYPE,
      );
    });

    it("should default the start offset and the record key", () => {
      const config = Kafka.Task.READ_SCHEMAS.config.parse({
        device: "dev",
        topic: "t",
        records: [{ key: "r", fields: [field] }],
      });
      expect(config.startOffset).toBe("latest");
      expect(config.records[0].recordKey).toBe("");
      expect(config.records[0].index).toBe("");
    });
  });

  describe("deployReadConfigZ", () => {
    const record = { key: "r", fields: [field] };
    const valid = { device: "dev", topic: "t", records: [record] };

    it("should accept a complete config", () => {
      expect(Kafka.Task.deployReadConfigZ.safeParse(valid).success).toBe(true);
    });

    it("should reject an empty topic", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({ ...valid, topic: "" });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toBe("Topic is required");
    });

    it("should reject a field without a pointer", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        records: [{ ...record, fields: [{ ...field, pointer: "" }] }],
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toBe("Pointer is required");
    });

    it("should reject a field whose pointer is not a JSON pointer", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        records: [{ ...record, fields: [{ ...field, pointer: "temperature" }] }],
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toBe(
        "Pointer must be a valid JSON pointer",
      );
    });

    it("should reject a timestamp field without a time format", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        records: [{ ...record, fields: [{ ...field, dataType: "timestamp" }] }],
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toBe(
        "Time format is required for timestamps",
      );
    });

    it("should accept a timestamp field with a time format", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        records: [
          {
            ...record,
            fields: [{ ...field, dataType: "timestamp", timeFormat: "unix_ms" }],
          },
        ],
      });
      expect(result.success).toBe(true);
    });

    it("should accept a disabled field without a pointer", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        records: [
          {
            ...record,
            fields: [
              field,
              { ...field, key: "f2", channel: 0, pointer: "", disabled: true },
            ],
          },
        ],
      });
      expect(result.success).toBe(true);
    });

    it("should reject a config with every field disabled", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        records: [{ ...record, fields: [{ ...field, disabled: true }] }],
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toBe(
        "At least one field must be enabled",
      );
    });

    it.each([
      ["", "Only one record can read any key"],
      ["pump", 'Record key "pump" is used by another record'],
    ])("should reject two records with the key %j", (recordKey, message) => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        records: [
          { ...record, recordKey },
          { ...record, key: "r2", recordKey },
        ],
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toBe(message);
      expect(result.error?.issues[0].path).toEqual(["records", 1, "recordKey"]);
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
