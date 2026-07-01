// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { HTTP } from "@/feature/http";

describe("HTTP Device Properties", () => {
  describe("propertiesZ", () => {
    it("should validate ZERO_PROPERTIES", () => {
      HTTP.Device.propertiesZ.parse(HTTP.Device.ZERO_PROPERTIES);
    });

    it("should validate a v1 config with none auth", () => {
      const config: HTTP.Device.Properties = {
        secure: true,
        verifySsl: false,
        timeoutMs: 500,
        auth: { type: "none" },
        healthCheck: HTTP.Device.ZERO_HEALTH_CHECK,
        write: {},
        read: {},
        version: 1,
      };
      const result = HTTP.Device.propertiesZ.parse(config);
      expect(result.auth.type).toBe("none");
    });

    it("should validate a v1 config with bearer auth", () => {
      const config: HTTP.Device.Properties = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "bearer", token: "my-token" },
        healthCheck: HTTP.Device.ZERO_HEALTH_CHECK,
        write: {},
        read: {},
        version: 1,
      };
      const result = HTTP.Device.propertiesZ.parse(config);
      expect(result.auth).toEqual({ type: "bearer", token: "my-token" });
    });

    it("should reject bearer auth with empty token", () => {
      const config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "bearer", token: "" },
        read: {},
        version: 1,
      };
      const result = HTTP.Device.propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should validate a v1 config with basic auth", () => {
      const config: HTTP.Device.Properties = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "basic", username: "user", password: "pass" },
        healthCheck: HTTP.Device.ZERO_HEALTH_CHECK,
        write: {},
        read: {},
        version: 1,
      };
      const result = HTTP.Device.propertiesZ.parse(config);
      expect(result.auth).toEqual({
        type: "basic",
        username: "user",
        password: "pass",
      });
    });

    it("should reject basic auth with empty username", () => {
      const config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "basic", username: "", password: "pass" },
        read: {},
        version: 1,
      };
      const result = HTTP.Device.propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should reject basic auth with empty password", () => {
      const config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "basic", username: "user", password: "" },
        read: {},
        version: 1,
      };
      const result = HTTP.Device.propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should validate a v1 config with api_key header auth", () => {
      const config: HTTP.Device.Properties = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: {
          type: "api_key",
          sendAs: "header",
          header: "X-API-Key",
          key: "secret",
        },
        healthCheck: HTTP.Device.ZERO_HEALTH_CHECK,
        write: {},
        read: {},
        version: 1,
      };
      const result = HTTP.Device.propertiesZ.parse(config);
      expect(result.auth).toEqual({
        type: "api_key",
        sendAs: "header",
        header: "X-API-Key",
        key: "secret",
      });
    });

    it("should reject api_key header auth with empty header", () => {
      const config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "api_key", sendAs: "header", header: "", key: "secret" },
        read: {},
        version: 1,
      };
      const result = HTTP.Device.propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should validate a v1 config with api_key query_param auth", () => {
      const config: HTTP.Device.Properties = {
        secure: false,
        verifySsl: false,
        timeoutMs: 200,
        auth: {
          type: "api_key",
          sendAs: "query_param",
          parameter: "api_key",
          key: "secret",
        },
        healthCheck: HTTP.Device.ZERO_HEALTH_CHECK,
        write: {},
        read: {},
        version: 1,
      };
      const result = HTTP.Device.propertiesZ.parse(config);
      expect(result.auth).toEqual({
        type: "api_key",
        sendAs: "query_param",
        parameter: "api_key",
        key: "secret",
      });
    });

    it("should reject api_key query_param auth with empty parameter", () => {
      const config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: {
          type: "api_key",
          sendAs: "query_param",
          parameter: "",
          key: "secret",
        },
        read: {},
        version: 1,
      };
      const result = HTTP.Device.propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should reject a negative timeout", () => {
      const config = {
        secure: true,
        verifySsl: true,
        timeoutMs: -1,
        auth: { type: "none" },
        read: {},
        version: 1,
      };
      const result = HTTP.Device.propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });
  });

  describe("healthCheckZ", () => {
    it("should validate a GET health check without response validation", () => {
      const config: HTTP.Device.Properties = {
        ...HTTP.Device.ZERO_PROPERTIES,
        healthCheck: {
          method: "GET",
          path: "/health",
          validateResponse: false,
        },
      };
      const result = HTTP.Device.propertiesZ.parse(config);
      expect(result.healthCheck.method).toBe("GET");
      expect(result.healthCheck.validateResponse).toBe(false);
    });

    it("should validate a GET health check with response validation", () => {
      const config: HTTP.Device.Properties = {
        ...HTTP.Device.ZERO_PROPERTIES,
        healthCheck: {
          method: "GET",
          path: "/health",
          validateResponse: true,
          response: {
            pointer: "/status",
            expectedValueType: "string",
            expectedValue: "ok",
          },
        },
      };
      const result = HTTP.Device.propertiesZ.parse(config);
      expect(result.healthCheck.validateResponse).toBe(true);
    });

    it("should validate a POST health check without response validation", () => {
      const config: HTTP.Device.Properties = {
        ...HTTP.Device.ZERO_PROPERTIES,
        healthCheck: {
          method: "POST",
          path: "/health",
          body: '{"check": true}',
          validateResponse: false,
        },
      };
      const result = HTTP.Device.propertiesZ.parse(config);
      expect(result.healthCheck.method).toBe("POST");
      expect(result.healthCheck.validateResponse).toBe(false);
    });

    it("should validate a POST health check with response validation", () => {
      const config: HTTP.Device.Properties = {
        ...HTTP.Device.ZERO_PROPERTIES,
        healthCheck: {
          method: "POST",
          path: "/health",
          body: '{"check": true}',
          validateResponse: true,
          response: {
            pointer: "/alive",
            expectedValueType: "boolean",
            expectedValue: true,
          },
        },
      };
      const result = HTTP.Device.propertiesZ.parse(config);
      expect(result.healthCheck.method).toBe("POST");
      expect(result.healthCheck.validateResponse).toBe(true);
    });

    it("should validate a POST health check without body", () => {
      const config: HTTP.Device.Properties = {
        ...HTTP.Device.ZERO_PROPERTIES,
        healthCheck: {
          method: "POST",
          path: "/health",
          validateResponse: false,
        },
      };
      const result = HTTP.Device.propertiesZ.parse(config);
      expect(result.healthCheck.method).toBe("POST");
    });

    it("should validate health check with headers and query params", () => {
      const config: HTTP.Device.Properties = {
        ...HTTP.Device.ZERO_PROPERTIES,
        healthCheck: {
          method: "GET",
          path: "/health",
          headers: [{ name: "Authorization", value: "Bearer token" }],
          queryParams: [{ parameter: "verbose", value: "true" }],
          validateResponse: false,
        },
      };
      const result = HTTP.Device.propertiesZ.parse(config);
      expect(result.healthCheck.path).toBe("/health");
    });

    it("should reject validateResponse true without response field", () => {
      const result = HTTP.Device.healthCheckZ.safeParse({
        method: "GET",
        path: "/health",
        validateResponse: true,
      });
      expect(result.success).toBe(false);
    });

    it("should reject validateResponse true with invalid pointer", () => {
      const result = HTTP.Device.healthCheckZ.safeParse({
        method: "GET",
        path: "/health",
        validateResponse: true,
        response: {
          pointer: "no-leading-slash",
          value: { expectedValueType: "string", expectedValue: "ok" },
        },
      });
      expect(result.success).toBe(false);
    });
  });

  describe("v0 migration", () => {
    it("should migrate v0 none auth to v1", () => {
      const v0Config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "none" },
        readIndexes: {},
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result.version).toBe(1);
      expect(result.auth).toEqual({ type: "none" });
    });

    it("should migrate v0 bearer auth to v1", () => {
      const v0Config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "bearer", token: "my-token" },
        readIndexes: {},
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result.version).toBe(1);
      expect(result.auth).toEqual({ type: "bearer", token: "my-token" });
    });

    it("should migrate v0 basic auth to v1", () => {
      const v0Config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "basic", username: "user", password: "pass" },
        readIndexes: {},
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result.version).toBe(1);
      expect(result.auth).toEqual({
        type: "basic",
        username: "user",
        password: "pass",
      });
    });

    it("should migrate v0 api_key auth to v1 header auth", () => {
      const v0Config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "api_key", header: "X-API-Key", key: "secret" },
        readIndexes: {},
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result.version).toBe(1);
      expect(result.auth).toEqual({
        type: "api_key",
        sendAs: "header",
        header: "X-API-Key",
        key: "secret",
      });
    });

    it("should migrate v0 none auth with queryParams to v1 api_key query_param auth", () => {
      const v0Config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "none" },
        queryParams: { api_key: "secret" },
        readIndexes: {},
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result.version).toBe(1);
      expect(result.auth).toEqual({
        type: "api_key",
        sendAs: "query_param",
        parameter: "api_key",
        key: "secret",
      });
    });

    it("should migrate v0 none auth with empty queryParams to v1 none auth", () => {
      const v0Config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "none" },
        queryParams: {},
        readIndexes: {},
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result.version).toBe(1);
      expect(result.auth).toEqual({ type: "none" });
    });

    it("should strip headers from v0 config", () => {
      const v0Config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "none" },
        headers: { "Content-Type": "application/json" },
        readIndexes: {},
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result).not.toHaveProperty("headers");
    });

    it("should apply defaults for omitted v0 fields", () => {
      const v0Config = {
        auth: { type: "none" },
        readIndexes: {},
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result.secure).toBe(true);
      expect(result.verifySsl).toBe(true);
      expect(result.timeoutMs).toBeGreaterThan(0);
    });

    it("should migrate v0 readIndexes to v1 read", () => {
      const v0Config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "none" },
        readIndexes: { "/api/data": 42, "/api/status": 99 },
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result.read).toEqual({
        "/api/data": { index: 42, channels: {} },
        "/api/status": { index: 99, channels: {} },
      });
      expect(result).not.toHaveProperty("readIndexes");
    });

    it("should migrate v0 empty readIndexes to v1 empty read", () => {
      const v0Config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "none" },
        readIndexes: {},
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result.read).toEqual({});
    });

    it("should set write to empty object on migration", () => {
      const v0Config = {
        auth: { type: "none" },
        readIndexes: {},
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result.write).toEqual({});
    });

    it("should set healthCheck to ZERO_HEALTH_CHECK on migration", () => {
      const v0Config = {
        auth: { type: "none" },
        readIndexes: {},
      };
      const result = HTTP.Device.propertiesZ.parse(v0Config);
      expect(result.healthCheck).toEqual(HTTP.Device.ZERO_HEALTH_CHECK);
    });
  });

  describe("headerEntryZ", () => {
    it("should validate a valid header entry", () => {
      const result = HTTP.Device.headerEntryZ.safeParse({
        name: "Accept",
        value: "text/html",
      });
      expect(result.success).toBe(true);
    });

    it("should reject a header entry missing name", () => {
      const result = HTTP.Device.headerEntryZ.safeParse({ value: "text/html" });
      expect(result.success).toBe(false);
    });

    it("should reject a header entry missing value", () => {
      const result = HTTP.Device.headerEntryZ.safeParse({ name: "Accept" });
      expect(result.success).toBe(false);
    });
  });

  describe("queryParamEntryZ", () => {
    it("should validate a valid query param entry", () => {
      const result = HTTP.Device.queryParamEntryZ.safeParse({
        parameter: "limit",
        value: "10",
      });
      expect(result.success).toBe(true);
    });

    it("should reject an entry missing parameter", () => {
      const result = HTTP.Device.queryParamEntryZ.safeParse({ value: "10" });
      expect(result.success).toBe(false);
    });

    it("should reject an entry missing value", () => {
      const result = HTTP.Device.queryParamEntryZ.safeParse({ parameter: "limit" });
      expect(result.success).toBe(false);
    });
  });

  describe("headersZ", () => {
    it("should validate a v1 header array", () => {
      const result = HTTP.Device.headersZ.parse([
        { name: "Accept", value: "application/json" },
      ]);
      expect(result).toEqual([{ name: "Accept", value: "application/json" }]);
    });

    it("should migrate a v0 header record to v1 array", () => {
      const result = HTTP.Device.headersZ.parse({
        Accept: "application/json",
        "X-Key": "val",
      });
      expect(result).toEqual([
        { name: "Accept", value: "application/json" },
        { name: "X-Key", value: "val" },
      ]);
    });

    it("should reject duplicate header names", () => {
      const result = HTTP.Device.headersZ.safeParse([
        { name: "Accept", value: "text/plain" },
        { name: "Accept", value: "application/json" },
      ]);
      expect(result.success).toBe(false);
    });

    it("should accept an empty array", () => {
      const result = HTTP.Device.headersZ.parse([]);
      expect(result).toEqual([]);
    });

    it("should skip duplicate check for empty name", () => {
      const result = HTTP.Device.headersZ.safeParse([
        { name: "", value: "a" },
        { name: "", value: "b" },
      ]);
      expect(result.success).toBe(true);
    });
  });

  describe("queryParamsZ", () => {
    it("should validate a v1 query param array", () => {
      const result = HTTP.Device.queryParamsZ.parse([
        { parameter: "limit", value: "10" },
      ]);
      expect(result).toEqual([{ parameter: "limit", value: "10" }]);
    });

    it("should migrate a v0 query param record to v1 array", () => {
      const result = HTTP.Device.queryParamsZ.parse({ limit: "10", offset: "20" });
      expect(result).toEqual([
        { parameter: "limit", value: "10" },
        { parameter: "offset", value: "20" },
      ]);
    });

    it("should reject duplicate parameter names", () => {
      const result = HTTP.Device.queryParamsZ.safeParse([
        { parameter: "limit", value: "10" },
        { parameter: "limit", value: "20" },
      ]);
      expect(result.success).toBe(false);
    });

    it("should accept an empty array", () => {
      const result = HTTP.Device.queryParamsZ.parse([]);
      expect(result).toEqual([]);
    });

    it("should skip duplicate check for empty parameter", () => {
      const result = HTTP.Device.queryParamsZ.safeParse([
        { parameter: "", value: "a" },
        { parameter: "", value: "b" },
      ]);
      expect(result.success).toBe(true);
    });
  });

  describe("checkDuplicateKeys", () => {
    it("should add an issue for duplicate keys", () => {
      const issues: unknown[] = [];
      const value = [{ name: "a" }, { name: "a" }];
      HTTP.Device.checkDuplicateKeys("name", "header")({ value, issues });
      expect(issues).toHaveLength(1);
    });

    it("should not add issues when keys are unique", () => {
      const issues: unknown[] = [];
      const value = [{ name: "a" }, { name: "b" }];
      HTTP.Device.checkDuplicateKeys("name", "header")({ value, issues });
      expect(issues).toHaveLength(0);
    });

    it("should handle undefined value", () => {
      const issues: unknown[] = [];
      HTTP.Device.checkDuplicateKeys("name", "header")({ value: undefined, issues });
      expect(issues).toHaveLength(0);
    });
  });
});
