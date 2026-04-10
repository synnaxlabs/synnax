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

import { id } from "@/id";
import { status } from "@/status";
import { TimeStamp } from "@/telem";

type CustomCrude = Partial<status.Crude<z.ZodRecord, "error">>;

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

    describe("custom errors (toStatus)", () => {
      class CustomError extends Error implements status.Custom {
        constructor(message: string) {
          super(message);
          this.name = "CustomError";
        }

        toStatus(): CustomCrude {
          return {
            message: "custom headline",
            description: "custom description body",
            details: { code: 42, hint: "try again" },
          };
        }
      }

      it("should use toStatus.message as the status message", () => {
        const s = status.fromException(new CustomError("raw"));
        expect(s.message).toBe("custom headline");
      });

      it("should use toStatus.description as the status description", () => {
        const s = status.fromException(new CustomError("raw"));
        expect(s.description).toBe("custom description body");
      });

      it("should merge toStatus.details into status.details", () => {
        const s = status.fromException(new CustomError("raw"));
        const details = s.details as Record<string, unknown>;
        expect(details.code).toBe(42);
        expect(details.hint).toBe("try again");
        expect(details.stack).toBeDefined();
        expect(details.error).toBeInstanceOf(Error);
      });

      it("should prefix toStatus.message when the caller provides a custom message", () => {
        const s = status.fromException(new CustomError("raw"), "Saving failed");
        expect(s.message).toBe("Saving failed: custom headline");
      });

      it("should fall through to the default path for errors without toStatus", () => {
        const s = status.fromException(new Error("plain"));
        expect(s.message).toBe("plain");
        expect(s.description).toBeUndefined();
      });

      it("should ignore toStatus if it throws", () => {
        class Bad extends Error {
          toStatus() {
            throw new Error("explode");
          }
        }
        const s = status.fromException(new Bad("fallback"));
        expect(s.message).toBe("fallback");
        expect(s.description).toBeUndefined();
      });

      it("should ignore toStatus if it returns a non-object", () => {
        class Bad extends Error {
          toStatus() {
            return "nope" as unknown as CustomCrude;
          }
        }
        const s = status.fromException(new Bad("fallback"));
        expect(s.message).toBe("fallback");
      });

      it("should ignore toStatus if it returns null", () => {
        class Bad extends Error {
          toStatus() {
            return null as unknown as CustomCrude;
          }
        }
        const s = status.fromException(new Bad("fallback"));
        expect(s.message).toBe("fallback");
      });

      it("should ignore individual fields with the wrong type", () => {
        class Bad extends Error {
          toStatus() {
            return {
              message: 42 as unknown as string,
              description: { not: "a string" } as unknown as string,
              details: "also not a record" as unknown as Record<string, unknown>,
            };
          }
        }
        const s = status.fromException(new Bad("fallback"));
        expect(s.message).toBe("fallback");
        expect(s.description).toBeUndefined();
      });

      it("should accept a partial return with only some valid fields", () => {
        class Partial extends Error {
          toStatus() {
            return { message: "only headline" };
          }
        }
        const s = status.fromException(new Partial("raw"));
        expect(s.message).toBe("only headline");
        expect(s.description).toBeUndefined();
      });

      it("should ignore toStatus if it is not a function", () => {
        const err = new Error("fallback") as Error & { toStatus: string };
        err.toStatus = "not a function";
        const s = status.fromException(err);
        expect(s.message).toBe("fallback");
      });
    });
  });

  describe("toString", () => {
    it("should format a basic status with variant and message", () => {
      const s = status.create({ variant: "success", message: "Operation completed" });
      const result = status.toString(s);
      expect(result).toBe("SUCCESS: Operation completed");
    });

    it("should include name when present and includeName is true", () => {
      const s = status.create({
        variant: "info",
        message: "System started",
        name: "SystemStatus",
      });
      const result = status.toString(s);
      expect(result).toBe("INFO [SystemStatus]: System started");
    });

    it("should exclude name when includeName is false", () => {
      const s = status.create({
        variant: "warning",
        message: "Low memory",
        name: "MemoryMonitor",
      });
      const result = status.toString(s, { includeName: false });
      expect(result).toBe("WARNING: Low memory");
    });

    it("should include timestamp when includeTimestamp is true", () => {
      const s = status.create({ variant: "error", message: "Failed" });
      const result = status.toString(s, { includeTimestamp: true });
      expect(result).toMatch(/^ERROR: Failed \(.+\)$/);
    });

    it("should format plain text description", () => {
      const s = status.create({
        variant: "error",
        message: "Connection failed",
        description: "Unable to reach server at localhost:9090",
      });
      const result = status.toString(s);
      expect(result).toBe(
        "ERROR: Connection failed\n\nDescription: Unable to reach server at localhost:9090",
      );
    });

    it("should pretty-print JSON description", () => {
      const s = status.create({
        variant: "error",
        message: "Validation failed",
        description: JSON.stringify({ field: "email", reason: "invalid format" }),
      });
      const result = status.toString(s);
      expect(result).toContain("ERROR: Validation failed");
      expect(result).toContain("Description:");
      expect(result).toContain('"field": "email"');
      expect(result).toContain('"reason": "invalid format"');
    });

    it("should include stack trace from exception details", () => {
      const error = new Error("Something went wrong");
      const s = status.fromException(error);
      const result = status.toString(s);
      expect(result).toContain("ERROR: Something went wrong");
      expect(result).toContain("Stack Trace:");
      expect(result).toContain("Error: Something went wrong");
    });

    it("should handle all variants", () => {
      const variants: status.Variant[] = [
        "success",
        "info",
        "warning",
        "error",
        "loading",
        "disabled",
      ];
      for (const variant of variants) {
        const s = status.create({ variant, message: "Test message" });
        const result = status.toString(s);
        expect(result).toBe(`${variant.toUpperCase()}: Test message`);
      }
    });

    it("should handle status with all optional fields", () => {
      const error = new Error("Full error");
      const s = status.fromException(error, "Custom message");
      s.name = "ErrorHandler";
      const result = status.toString(s, { includeTimestamp: true });
      expect(result).toContain("ERROR [ErrorHandler]: Custom message");
      expect(result).toContain("Description: Full error");
      expect(result).toContain("Stack Trace:");
    });

    it("should not include name brackets when name is empty string", () => {
      const s = status.create({ variant: "success", message: "Done" });
      const result = status.toString(s);
      expect(result).toBe("SUCCESS: Done");
      expect(result).not.toContain("[]");
    });

    it("should include extra details beyond stack trace", () => {
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const detailsSchema = z.object({
        statusCode: z.number(),
        endpoint: z.string(),
        requestId: z.string(),
      });
      const s = status.create<typeof detailsSchema>({
        variant: "error",
        message: "Request failed",
        details: {
          statusCode: 404,
          endpoint: "/api/users",
          requestId: "abc-123",
        },
      });
      const result = status.toString(s);
      expect(result).toContain("ERROR: Request failed");
      expect(result).toContain("Details:");
      expect(result).toContain('"statusCode": 404');
      expect(result).toContain('"endpoint": "/api/users"');
      expect(result).toContain('"requestId": "abc-123"');
    });

    it("should include both stack trace and extra details separately", () => {
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const detailsSchema = z.object({
        stack: z.string(),
        query: z.string(),
        duration: z.number(),
      });
      const s = status.create<typeof detailsSchema>({
        variant: "error",
        message: "Database error",
        details: {
          stack: "Error: Database error\n    at query (db.ts:10)",
          query: "SELECT * FROM users",
          duration: 1500,
        },
      });
      const result = status.toString(s);
      expect(result).toContain("ERROR: Database error");
      expect(result).toContain("Stack Trace:\nError: Database error");
      expect(result).toContain("Details:");
      expect(result).toContain('"query": "SELECT * FROM users"');
      expect(result).toContain('"duration": 1500');
      expect(result).not.toContain('"stack"');
    });

    it("should exclude error object from details output", () => {
      const error = new Error("Test error");
      const s = status.fromException(error);
      const result = status.toString(s);
      expect(result).toContain("Stack Trace:");
      expect(result).not.toContain('"error"');
    });

    it("should not include Details section when only stack and error are present", () => {
      const error = new Error("Test error");
      const s = status.fromException(error);
      const result = status.toString(s);
      expect(result).toContain("Stack Trace:");
      expect(result).not.toContain("Details:");
    });

    it("should not error when the `name` field is accidentally undefined", () => {
      const s = status.create({ message: "cat", variant: "success" });
      const result = status.toString(s);
      expect(result).toContain("cat");
    });

    describe("custom errors (toStatus)", () => {
      class CustomError extends Error implements status.Custom {
        constructor() {
          super("raw");
          this.name = "CustomError";
        }

        toStatus(): CustomCrude {
          return {
            message: "custom headline",
            description: "line one\nline two\nline three",
            details: { code: 42 },
          };
        }
      }

      it("should render the toStatus headline as the status message", () => {
        const s = status.fromException(new CustomError());
        expect(status.toString(s)).toContain("ERROR: custom headline");
      });

      it("should render a multi-line description on its own block", () => {
        const out = status.toString(status.fromException(new CustomError()));
        expect(out).toContain("Description:\nline one\nline two\nline three");
      });

      it("should include merged details under the Details section", () => {
        const out = status.toString(status.fromException(new CustomError()));
        expect(out).toContain("Details:");
        expect(out).toContain('"code": 42');
      });
    });
  });
});
