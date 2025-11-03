// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { csv } from "@/csv";

describe("csv", () => {
  describe("maybeEscapeField", () => {
    it("shouldn't escape fields without special characters", () => {
      expect(csv.maybeEscapeField("hello")).toBe("hello");
      expect(csv.maybeEscapeField("123")).toBe("123");
      expect(csv.maybeEscapeField("Call me Ishmael")).toBe("Call me Ishmael");
    });
    it("should escape fields with special characters", () => {
      expect(csv.maybeEscapeField("hello,world")).toBe('"hello,world"');
      expect(csv.maybeEscapeField("hello\nworld")).toBe('"hello\nworld"');
      expect(csv.maybeEscapeField("hello\r\nworld")).toBe('"hello\r\nworld"');
      expect(csv.maybeEscapeField('"hello"')).toBe('"""hello"""');
    });
  });
});
