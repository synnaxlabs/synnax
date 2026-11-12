// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";

describe("kind", () => {
  describe("kindOf", () => {
    it.each([
      ["float32", "float"],
      ["float64", "float"],
      ["int8", "integer"],
      ["int64", "integer"],
      ["uint8", "unsigned"],
      ["uint64", "unsigned"],
      ["boolean", "boolean"],
      ["timestamp", "timestamp"],
      ["json", "json"],
      ["uuid", "uuid"],
      ["bytes", "bytes"],
      ["string", "string"],
    ] as const)("should map %s to %s", (dataType, kind) => {
      expect(Task.Payload.kindOf(dataType)).toBe(kind);
    });
  });

  describe("dataTypeOf", () => {
    it.each([
      ["float", "float64"],
      ["integer", "int64"],
      ["unsigned", "uint64"],
      ["string", "string"],
      ["boolean", "boolean"],
      ["timestamp", "timestamp"],
      ["json", "json"],
      ["uuid", "uuid"],
      ["bytes", "bytes"],
    ] as const)("should give a %s field a %s channel by default", (type, dataType) => {
      expect(Task.Payload.dataTypeOf({ type })).toBe(dataType);
    });

    it("should use the data type that a field states", () => {
      expect(Task.Payload.dataTypeOf({ type: "integer", dataType: "int16" })).toBe(
        "int16",
      );
    });
  });
});
