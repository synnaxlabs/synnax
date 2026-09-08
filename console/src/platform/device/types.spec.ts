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

import { Device } from "@/platform/device";

describe("device types", () => {
  describe("nameZ", () => {
    it("should accept a non-empty name", () => {
      expect(z.validate(Device.nameZ, "Sensor 1")).toBe(true);
    });

    it("should reject an empty name", () => {
      const result = Device.nameZ.safeParse("");
      expect(result.success).toBe(false);
      if (!result.success)
        expect(result.error.issues[0].message).toEqual("Name is required");
    });
  });

  describe("identifierZ", () => {
    it("should accept a valid identifier", () => {
      expect(z.validate(Device.identifierZ, "dev_1")).toBe(true);
    });

    it("should accept a two-character identifier at the lower bound", () => {
      expect(z.validate(Device.identifierZ, "ab")).toBe(true);
    });

    it("should accept a twelve-character identifier at the upper bound", () => {
      expect(z.validate(Device.identifierZ, "abcdefghijkl")).toBe(true);
    });

    it("should reject an identifier shorter than two characters", () => {
      const result = Device.identifierZ.safeParse("a");
      expect(result.success).toBe(false);
      if (!result.success)
        expect(result.error.issues[0].message).toEqual(
          "Identifier must be between 2-12 characters",
        );
    });

    it("should reject an identifier longer than twelve characters", () => {
      expect(z.validate(Device.identifierZ, "abcdefghijklm")).toBe(false);
    });

    it("should reject an identifier that does not start with a letter", () => {
      const result = Device.identifierZ.safeParse("1abc");
      expect(result.success).toBe(false);
      if (!result.success)
        expect(result.error.issues[0].message).toContain("must start with a letter");
    });

    it("should reject an identifier containing spaces or dashes", () => {
      expect(z.validate(Device.identifierZ, "dev-1")).toBe(false);
      expect(z.validate(Device.identifierZ, "dev 1")).toBe(false);
    });
  });

  describe("commandStatePairZ", () => {
    it("should parse a valid command/state pair", () => {
      expect(Device.commandStatePairZ.parse({ command: 3, state: 4 })).toEqual({
        command: 3,
        state: 4,
      });
    });

    it("should reject a pair missing a channel key", () => {
      expect(z.validate(Device.commandStatePairZ, { command: 3 })).toBe(false);
    });
  });

  describe("ZERO_COMMAND_STATE_PAIR", () => {
    it("should be a valid command/state pair with zeroed keys", () => {
      expect(Device.commandStatePairZ.parse(Device.ZERO_COMMAND_STATE_PAIR)).toEqual({
        command: 0,
        state: 0,
      });
    });
  });
});
