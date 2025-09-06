// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  errorResult,
  nullClientResult,
  pendingResult,
  successResult,
} from "@/flux/result";

interface TestState {
  id: string;
  name: string;
  value: number;
}

describe("result", () => {
  describe("pendingResult", () => {
    it("should create a loading result with correct structure", () => {
      const result = pendingResult<TestState>("user", "fetch", undefined);

      expect(result.variant).toBe("loading");
      expect(result.status.message).toBe("Fetch user");
      expect(result.data).toBeUndefined();
    });

    it("should capitalize the operation name", () => {
      const result = pendingResult<TestState>("channel", "create", undefined);

      expect(result.status.message).toBe("Create channel");
    });

    it("should handle complex operation names", () => {
      const result = pendingResult<TestState>(
        "database connection",
        "establish",
        undefined,
      );

      expect(result.status.message).toBe("Establish database connection");
    });
  });

  describe("successResult", () => {
    it("should create a success result with correct structure", () => {
      const testData: TestState = {
        id: "123",
        name: "Test User",
        value: 42,
      };

      const result = successResult<TestState>("user", "fetch", testData);

      expect(result.variant).toBe("success");
      expect(result.status.message).toBe("Fetch user");
      expect(result.data).toEqual(testData);
    });

    it("should preserve the exact data provided", () => {
      const complexData: TestState = {
        id: "456",
        name: "Complex User",
        value: 100,
      };

      const result = successResult<TestState>("entity", "save", complexData);

      expect(result.data).toBe(complexData);
      expect(result.data?.id).toBe("456");
      expect(result.data?.name).toBe("Complex User");
      expect(result.data?.value).toBe(100);
    });

    it("should capitalize the operation name", () => {
      const testData: TestState = {
        id: "789",
        name: "Test",
        value: 0,
      };

      const result = successResult<TestState>("item", "update", testData);

      expect(result.status.message).toBe("Update item");
    });
  });

  describe("errorResult", () => {
    it("should create an error result with correct structure", () => {
      const testError = new Error("Test error");

      const result = errorResult<TestState>("user", "fetch", testError);

      expect(result.variant).toBe("error");
      expect(result.status.message).toBe("Failed to fetch user");
      expect(result.data).toBeUndefined();
    });

    it("should include exception details when error is an Error object", () => {
      const error = new Error("Database connection timeout");
      const result = errorResult<TestState>("connection", "establish", error);
      expect(result.variant).toBe("error");
      expect(result.status.description).toBe("Database connection timeout");
    });
  });

  describe("nullClientResult", () => {
    it("should create an error result with DisconnectedError", () => {
      const result = nullClientResult<TestState>("user", "fetch");

      expect(result.variant).toBe("disabled");
      expect(result.status.message).toBe("Failed to fetch user");
      expect(result.data).toBeUndefined();
    });

    it("should include correct disconnection message", () => {
      const result = nullClientResult<TestState>("channel", "create");

      expect(result.status.description).toBe(
        "Cannot create channel because no cluster is connected.",
      );
    });

    it("should handle different operation names", () => {
      const result = nullClientResult<TestState>("database", "query");

      expect(result.status.message).toBe("Failed to query database");
      expect(result.status.description).toBe(
        "Cannot query database because no cluster is connected.",
      );
    });

    it("should maintain consistent structure with other error results", () => {
      const result = nullClientResult<TestState>("service", "start");
      expect(result.variant).toBe("disabled");
      expect(result.data).toBeUndefined();
      expect(result.status.message).toBeDefined();
      expect(result.status.description).toBeDefined();
    });
  });
});
