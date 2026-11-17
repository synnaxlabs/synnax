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
