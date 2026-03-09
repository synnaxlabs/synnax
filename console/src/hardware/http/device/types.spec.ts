// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  healthCheckZ,
  type Properties,
  propertiesZ,
  ZERO_HEALTH_CHECK,
  ZERO_PROPERTIES,
} from "@/hardware/http/device/types";

describe("HTTP Device Properties", () => {
  describe("propertiesZ", () => {
    it("should validate ZERO_PROPERTIES", () => {
      propertiesZ.parse(ZERO_PROPERTIES);
    });

    it("should validate a v1 config with none auth", () => {
      const config: Properties = {
        secure: true,
        verifySsl: false,
        timeoutMs: 500,
        auth: { type: "none" },
        healthCheck: ZERO_HEALTH_CHECK,
        readIndexes: {},
        version: 1,
      };
      const result = propertiesZ.parse(config);
      expect(result.auth.type).toBe("none");
    });

    it("should validate a v1 config with bearer auth", () => {
      const config: Properties = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "bearer", token: "my-token" },
        healthCheck: ZERO_HEALTH_CHECK,
        readIndexes: {},
        version: 1,
      };
      const result = propertiesZ.parse(config);
      expect(result.auth).toEqual({ type: "bearer", token: "my-token" });
    });

    it("should reject bearer auth with empty token", () => {
      const config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "bearer", token: "" },
        readIndexes: {},
        version: 1,
      };
      const result = propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should validate a v1 config with basic auth", () => {
      const config: Properties = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "basic", username: "user", password: "pass" },
        healthCheck: ZERO_HEALTH_CHECK,
        readIndexes: {},
        version: 1,
      };
      const result = propertiesZ.parse(config);
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
        readIndexes: {},
        version: 1,
      };
      const result = propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should reject basic auth with empty password", () => {
      const config = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: { type: "basic", username: "user", password: "" },
        readIndexes: {},
        version: 1,
      };
      const result = propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should validate a v1 config with api_key header auth", () => {
      const config: Properties = {
        secure: true,
        verifySsl: true,
        timeoutMs: 100,
        auth: {
          type: "api_key",
          sendAs: "header",
          header: "X-API-Key",
          key: "secret",
        },
        healthCheck: ZERO_HEALTH_CHECK,
        readIndexes: {},
        version: 1,
      };
      const result = propertiesZ.parse(config);
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
        readIndexes: {},
        version: 1,
      };
      const result = propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should validate a v1 config with api_key query_param auth", () => {
      const config: Properties = {
        secure: false,
        verifySsl: false,
        timeoutMs: 200,
        auth: {
          type: "api_key",
          sendAs: "query_param",
          parameter: "api_key",
          key: "secret",
        },
        healthCheck: ZERO_HEALTH_CHECK,
        readIndexes: {},
        version: 1,
      };
      const result = propertiesZ.parse(config);
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
        readIndexes: {},
        version: 1,
      };
      const result = propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should reject a negative timeout", () => {
      const config = {
        secure: true,
        verifySsl: true,
        timeoutMs: -1,
        auth: { type: "none" },
        readIndexes: {},
        version: 1,
      };
      const result = propertiesZ.safeParse(config);
      expect(result.success).toBe(false);
    });
  });

  describe("healthCheckZ", () => {
    it("should validate a GET health check without response validation", () => {
      const config: Properties = {
        ...ZERO_PROPERTIES,
        healthCheck: {
          method: "GET",
          path: "/health",
          validateResponse: false,
        },
      };
      const result = propertiesZ.parse(config);
      expect(result.healthCheck.method).toBe("GET");
      expect(result.healthCheck.validateResponse).toBe(false);
    });

    it("should validate a GET health check with response validation", () => {
      const config: Properties = {
        ...ZERO_PROPERTIES,
        healthCheck: {
          method: "GET",
          path: "/health",
          validateResponse: true,
          response: {
            pointer: "/status",
            value: { expectedValueType: "string", expectedValue: "ok" },
          },
        },
      };
      const result = propertiesZ.parse(config);
      expect(result.healthCheck.validateResponse).toBe(true);
    });

    it("should validate a POST health check without response validation", () => {
      const config: Properties = {
        ...ZERO_PROPERTIES,
        healthCheck: {
          method: "POST",
          path: "/health",
          body: '{"check": true}',
          validateResponse: false,
        },
      };
      const result = propertiesZ.parse(config);
      expect(result.healthCheck.method).toBe("POST");
      expect(result.healthCheck.validateResponse).toBe(false);
    });

    it("should validate a POST health check with response validation", () => {
      const config: Properties = {
        ...ZERO_PROPERTIES,
        healthCheck: {
          method: "POST",
          path: "/health",
          body: '{"check": true}',
          validateResponse: true,
          response: {
            pointer: "/alive",
            value: { expectedValueType: "boolean", expectedValue: true },
          },
        },
      };
      const result = propertiesZ.parse(config);
      expect(result.healthCheck.method).toBe("POST");
      expect(result.healthCheck.validateResponse).toBe(true);
    });

    it("should validate a POST health check without body", () => {
      const config: Properties = {
        ...ZERO_PROPERTIES,
        healthCheck: {
          method: "POST",
          path: "/health",
          validateResponse: false,
        },
      };
      const result = propertiesZ.parse(config);
      expect(result.healthCheck.method).toBe("POST");
    });

    it("should validate health check with headers and query params", () => {
      const config: Properties = {
        ...ZERO_PROPERTIES,
        healthCheck: {
          method: "GET",
          path: "/health",
          headers: { Authorization: "Bearer token" },
          queryParams: { verbose: "true" },
          validateResponse: false,
        },
      };
      const result = propertiesZ.parse(config);
      expect(result.healthCheck.path).toBe("/health");
    });

    it("should reject a health check with empty path", () => {
      const result = healthCheckZ.safeParse({
        method: "GET",
        path: "",
        validateResponse: false,
      });
      expect(result.success).toBe(false);
    });

    it("should reject validateResponse true without response field", () => {
      const result = healthCheckZ.safeParse({
        method: "GET",
        path: "/health",
        validateResponse: true,
      });
      expect(result.success).toBe(false);
    });

    it("should reject validateResponse true with invalid pointer", () => {
      const result = healthCheckZ.safeParse({
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
      const result = propertiesZ.parse(v0Config);
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
      const result = propertiesZ.parse(v0Config);
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
      const result = propertiesZ.parse(v0Config);
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
      const result = propertiesZ.parse(v0Config);
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
      const result = propertiesZ.parse(v0Config);
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
      const result = propertiesZ.parse(v0Config);
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
      const result = propertiesZ.parse(v0Config);
      expect(result).not.toHaveProperty("headers");
    });

    it("should apply defaults for omitted v0 fields", () => {
      const v0Config = {
        auth: { type: "none" },
        readIndexes: {},
      };
      const result = propertiesZ.parse(v0Config);
      expect(result.secure).toBe(true);
      expect(result.verifySsl).toBe(true);
      expect(result.timeoutMs).toBeGreaterThan(0);
    });
  });
});
