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
        fields: [field],
      });
      expect(config.startOffset).toBe("latest");
      expect(config.fields[0].recordKey).toBe("");
    });
  });

  describe("deployReadConfigZ", () => {
    const valid = { device: "dev", topic: "t", fields: [field] };

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
        fields: [{ ...field, pointer: "" }],
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toBe("Pointer is required");
    });

    it("should reject a config with every field disabled", () => {
      const result = Kafka.Task.deployReadConfigZ.safeParse({
        ...valid,
        fields: [{ ...field, disabled: true }],
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toBe(
        "At least one field must be enabled",
      );
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

    it("should reject an empty value pointer", () => {
      const result = Kafka.Task.deployWriteConfigZ.safeParse({
        ...valid,
        record: { valuePointer: "" },
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toBe("Value pointer is required");
    });
  });
});
