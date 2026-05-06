// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type errors } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import {
  type Adder,
  createAsyncErrorHandler,
  createErrorHandler,
} from "@/status/aether/errorHandler";

describe("errorHandler", () => {
  describe("checkSkip", () => {
    it("should skip errors that match single matcher", () => {
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const error = new Error("test error");
      const matcher: errors.Matchable = {
        matches: (e) => e === error,
      };
      handler(error, "message", matcher);
      expect(mockAdder).not.toHaveBeenCalled();
    });

    it("should skip errors that match any matcher in array", () => {
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const error = new Error("test error");
      const matchers: errors.Matchable[] = [
        { matches: () => false },
        { matches: (e) => e === error },
      ];
      handler(error, "message", matchers);
      expect(mockAdder).not.toHaveBeenCalled();
    });

    it("should not skip errors that don't match", () => {
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const error = new Error("test error");
      const matcher: errors.Matchable = {
        matches: () => false,
      };
      handler(error, "message", matcher);
      expect(mockAdder).toHaveBeenCalledWith(expect.any(Object));
    });
  });

  describe("createErrorHandler", () => {
    it("should handle direct exceptions", () => {
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const error = new Error("test error");

      handler(error, "custom message");

      expect(mockAdder).toHaveBeenCalledWith(
        expect.objectContaining({
          message: "custom message",
        }),
      );
    });

    it("should handle synchronous functions that throw", () => {
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const error = new Error("sync error");
      const func = () => {
        throw error;
      };

      handler(func, "sync error message");

      expect(mockAdder).toHaveBeenCalledWith(
        expect.objectContaining({
          message: "sync error message",
        }),
      );
    });

    it("should handle synchronous functions that don't throw", () => {
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const func = () => {
        // no error
      };

      handler(func);

      expect(mockAdder).not.toHaveBeenCalled();
    });

    it("should handle async functions that reject", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const error = new Error("async error");
      const func = async () => {
        throw error;
      };

      handler(func, "async error message");

      // Wait for the async operation to complete
      await new Promise((resolve) => setTimeout(resolve, 0));

      expect(mockAdder).toHaveBeenCalledWith(
        expect.objectContaining({
          message: "async error message",
        }),
      );
    });

    it("should handle async functions that resolve", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const func = async () => {
        // no error
      };

      handler(func);

      await new Promise((resolve) => setTimeout(resolve, 0));

      expect(mockAdder).not.toHaveBeenCalled();
    });

    it("should not add status if error is skipped", () => {
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const error = new Error("skippable");
      const skip: errors.Matchable = {
        matches: (e) => e instanceof Error && e.message === "skippable",
      };

      handler(error, "message", skip);

      expect(mockAdder).not.toHaveBeenCalled();
    });

    it("should log errors to console", () => {
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const error = new Error("console test");

      handler(error);

      expect(consoleSpy).toHaveBeenCalled();
      consoleSpy.mockRestore();
    });
  });

  describe("createAsyncErrorHandler", () => {
    it("should handle direct exceptions", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      const error = new Error("test error");

      await handler(error, "custom message");

      expect(mockAdder).toHaveBeenCalledWith(
        expect.objectContaining({
          message: "custom message",
        }),
      );
    });

    it("should handle synchronous functions that throw", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      const error = new Error("sync error");
      const func = () => {
        throw error;
      };

      await handler(func, "sync error message");

      expect(mockAdder).toHaveBeenCalledWith(
        expect.objectContaining({
          message: "sync error message",
        }),
      );
    });

    it("should handle synchronous functions that don't throw", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      const func = () => {
        // no error
      };

      await handler(func);

      expect(mockAdder).not.toHaveBeenCalled();
    });

    it("should handle async functions that reject", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      const error = new Error("async error");
      const func = async () => {
        throw error;
      };

      await handler(func, "async error message");

      expect(mockAdder).toHaveBeenCalledWith(
        expect.objectContaining({
          message: "async error message",
        }),
      );
    });

    it("should handle async functions that resolve", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      const func = async () => {
        // no error
      };

      await handler(func);

      expect(mockAdder).not.toHaveBeenCalled();
    });

    it("should not add status if error is skipped", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      const error = new Error("skippable");
      const skip: errors.Matchable = {
        matches: (e) => e instanceof Error && e.message === "skippable",
      };

      await handler(error, "message", skip);

      expect(mockAdder).not.toHaveBeenCalled();
    });

    it("should handle multiple skip matchers", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      const error = new TypeError("type error");
      const skips: errors.Matchable[] = [
        { matches: (e) => e instanceof RangeError },
        { matches: (e) => e instanceof TypeError },
      ];

      await handler(error, "message", skips);

      expect(mockAdder).not.toHaveBeenCalled();
    });

    it("should log errors to console", async () => {
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      const error = new Error("console test");

      await handler(error);

      expect(consoleSpy).toHaveBeenCalled();
      consoleSpy.mockRestore();
    });

    it("should await promise returned by function", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      let resolved = false;
      const func = async () => {
        await new Promise((resolve) => setTimeout(resolve, 10));
        resolved = true;
      };

      await handler(func);

      expect(resolved).toBe(true);
      expect(mockAdder).not.toHaveBeenCalled();
    });

    it("should handle promise rejection properly", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      const error = new Error("promise rejection");
      const func = async () => {
        await new Promise((_, reject) => setTimeout(() => reject(error), 10));
      };

      await handler(func, "rejection message");

      expect(mockAdder).toHaveBeenCalledWith(
        expect.objectContaining({
          message: "rejection message",
        }),
      );
    });
    it("should rehrow exceptions that are not errors - undefined", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      const func = () => {
        // eslint-disable-next-line @typescript-eslint/only-throw-error
        throw undefined;
      };
      await expect(handler(func, "undefined error")).rejects.toThrow("undefined error");
      expect(mockAdder).not.toHaveBeenCalled();
    });

    it("should rethrow exceptions that are not errors - string", async () => {
      const mockAdder: Adder = vi.fn();
      const handler = createAsyncErrorHandler(mockAdder);
      const func = () => {
        // eslint-disable-next-line @typescript-eslint/only-throw-error
        throw "dog";
      };
      await expect(handler(func, "string error")).rejects.toThrow("dog");
      expect(mockAdder).not.toHaveBeenCalled();
    });
  });

  describe("parseException", () => {
    it("should create status from exception", () => {
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const error = new Error("parse test");

      handler(error, "custom message");

      expect(mockAdder).toHaveBeenCalledWith(
        expect.objectContaining({
          message: "custom message",
        }),
      );
    });
  });

  describe("Adder interface", () => {
    it("should accept status with data", () => {
      const mockAdder: Adder = vi.fn();
      const handler = createErrorHandler(mockAdder);
      const error = new Error("test");

      handler(error, "message");

      expect(mockAdder).toHaveBeenCalledWith(expect.any(Object));
      const call = (mockAdder as any).mock.calls[0][0];
      expect(call).toHaveProperty("message");
    });

    it("should be called with proper status structure", () => {
      let capturedStatus: any;
      const mockAdder: Adder = (spec) => {
        capturedStatus = spec;
      };
      const handler = createErrorHandler(mockAdder);
      const error = new Error("structured error");

      handler(error, "structured message");

      expect(capturedStatus).toBeDefined();
      expect(capturedStatus.message).toBe("structured message");
    });
  });
});
