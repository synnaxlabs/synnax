// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { id } from "@/id";
import { status } from "@/status";
import { TimeStamp } from "@/telem";

describe("status", () => {
  describe("create", () => {
    it("should create a status", () => {
      const s = status.create({ variant: "success", message: "test" });
      expect(s.key).toHaveLength(id.LENGTH);
      expect(s.time).toBeInstanceOf(TimeStamp);
      expect(s.time.beforeEq(TimeStamp.now())).toBe(true);
    });
  });

  describe("keepVariants", () => {
    it("should return undefined when variant is null", () => {
      expect(status.keepVariants(undefined, "success")).toBeUndefined();
    });

    it("should return undefined when variant is not in keep list", () => {
      expect(status.keepVariants("error", "success")).toBeUndefined();
      expect(status.keepVariants("error", ["success", "info"])).toBeUndefined();
    });

    it("should return variant when it matches single keep variant", () => {
      expect(status.keepVariants("success", "success")).toBe("success");
    });

    it("should return variant when it is in keep array", () => {
      expect(status.keepVariants("success", ["success", "info"])).toBe("success");
      expect(status.keepVariants("info", ["success", "info"])).toBe("info");
    });

    it("should return undefined when keep is empty array", () => {
      expect(status.keepVariants("success", [])).toBeUndefined();
    });
  });

  describe("removeVariants", () => {
    it("should return undefined when variant is null", () => {
      expect(status.removeVariants(undefined, "success")).toBeUndefined();
    });

    it("should return undefined when variant matches single remove variant", () => {
      expect(status.removeVariants("success", "success")).toBeUndefined();
    });

    it("should return undefined when variant is in remove array", () => {
      expect(status.removeVariants("success", ["success", "error"])).toBeUndefined();
      expect(status.removeVariants("error", ["success", "error"])).toBeUndefined();
    });

    it("should return variant when it does not match single remove variant", () => {
      expect(status.removeVariants("success", "error")).toBe("success");
    });

    it("should return variant when it is not in remove array", () => {
      expect(status.removeVariants("warning", ["success", "error"])).toBe("warning");
      expect(status.removeVariants("info", ["success", "error"])).toBe("info");
    });

    it("should return variant when remove is empty array", () => {
      expect(status.removeVariants("success", [])).toBe("success");
    });
  });

  describe("fromException", () => {
    it("should create an error status from an Error instance", () => {
      const error = new Error("Something went wrong");
      const s = status.fromException(error);

      expect(s.variant).toBe("error");
      expect(s.message).toBe("Something went wrong");
      expect(s.description).toBeUndefined();
      expect(s.details.error).toBe(error);
      expect(s.details.stack).toBe(error.stack ?? "");
    });

    it("should use custom message and move error message to description", () => {
      const error = new Error("Original error");
      const s = status.fromException(error, "Custom message");

      expect(s.variant).toBe("error");
      expect(s.message).toBe("Custom message");
      expect(s.description).toBe("Original error");
      expect(s.details.error).toBe(error);
      expect(s.details.stack).toBe(error.stack ?? "");
    });

    it("should handle errors without stack trace", () => {
      const error = new Error("No stack");
      error.stack = undefined;
      const s = status.fromException(error);

      expect(s.details.stack).toBe("");
      expect(s.details.error).toBe(error);
    });

    it("should throw when exception is not an Error instance", () => {
      const notAnError = "just a string";
      expect(() => status.fromException(notAnError)).toThrow("just a string");
    });

    it("should include valid key and timestamp", () => {
      const error = new Error("Test error");
      const s = status.fromException(error);

      expect(s.key).toHaveLength(id.LENGTH);
      expect(s.time).toBeInstanceOf(TimeStamp);
      expect(s.time.beforeEq(TimeStamp.now())).toBe(true);
    });

    it("should conform to exceptionDetailsSchema", () => {
      const error = new Error("Test error");
      const s = status.fromException(error);

      const result = status.exceptionDetailsSchema.safeParse(s.details);
      expect(result.success).toBe(true);
    });
  });
});
