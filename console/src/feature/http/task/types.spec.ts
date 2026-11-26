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

import { HTTP } from "@/feature/http";

/** The ICD and message of a write endpoint that sends one. */
const SENDS = {
  icd: "8c1f2e6a-4b7d-4c3e-9f10-2a5b6c7d8e9f",
  message: { message: "1d2e3f4a-5b6c-4d7e-8f90-a1b2c3d4e5f6", bindings: [] },
};

describe("HTTP Task Types", () => {
  describe("READ_SCHEMAS", () => {
    it("should validate the type literal", () => {
      expect(HTTP.Task.READ_SCHEMAS.type.parse(HTTP.Task.READ_TYPE)).toBe(
        HTTP.Task.READ_TYPE,
      );
    });

    it("should validate a config with a GET endpoint", () => {
      const config = {
        device: "dev-001",
        dataSavingDisabled: false,
        autoStart: false,
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api/data",
          },
        ],
      };
      const result = HTTP.Task.READ_SCHEMAS.config.parse(config);
      expect(result.endpoints).toHaveLength(1);
      expect(result.endpoints[0].method).toBe("GET");
    });

    it("should validate a config with a POST endpoint and body", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api/query",
            body: '{"query": "latest"}',
          },
        ],
      };
      const result = HTTP.Task.READ_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].method).toBe("POST");
    });

    it("should default an endpoint without a body to an empty body", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api/data",
          },
        ],
      };
      const result = HTTP.Task.READ_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].body).toBe("");
    });

    it("should validate a config with endpoint headers and query params", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api/data",
            headers: [{ name: "Accept", value: "application/json" }],
            queryParams: [{ parameter: "limit", value: "100" }],
          },
        ],
      };
      const result = HTTP.Task.READ_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].headers).toEqual([
        { name: "Accept", value: "application/json" },
      ]);
      expect(result.endpoints[0].queryParams).toEqual([
        { parameter: "limit", value: "100" },
      ]);
    });

    it("should reject a non-positive rate", () => {
      const config = {
        device: "dev-001",
        rate: 0,
        endpoints: [],
      };
      expect(HTTP.Task.deployReadConfigZ.validate(config)).toBe(false);
    });

    it("should reject a negative rate", () => {
      const config = {
        device: "dev-001",
        rate: -1,
        endpoints: [],
      };
      expect(HTTP.Task.deployReadConfigZ.validate(config)).toBe(false);
    });

    it("should validate statusData as running/message object", () => {
      HTTP.Task.READ_SCHEMAS.statusData.parse({ running: true, message: "ok" });
    });

    it("should validate statusData as null", () => {
      HTTP.Task.READ_SCHEMAS.statusData.parse(null);
    });

    it("should validate statusData as undefined", () => {
      expect(z.validate(HTTP.Task.READ_SCHEMAS.statusData, undefined)).toBe(true);
    });
  });

  describe("read endpoint", () => {
    it("should drop the fields and index of an endpoint from before ICDs", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            index: "f1",
            fields: [{ key: "f1", pointer: "/v", channel: 1 }],
          },
        ],
      };
      const [endpoint] = HTTP.Task.READ_SCHEMAS.config.parse(config).endpoints;
      expect(endpoint).not.toHaveProperty("fields");
      expect(endpoint).not.toHaveProperty("index");
    });

    it("should reject duplicate header names on an endpoint at deploy", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            headers: [
              { name: "X-Key", value: "a" },
              { name: "X-Key", value: "b" },
            ],
          },
        ],
      };
      expect(HTTP.Task.deployReadConfigZ.validate(config)).toBe(false);
    });

    it("should reject duplicate query parameter names on an endpoint at deploy", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            queryParams: [
              { parameter: "key", value: "a" },
              { parameter: "key", value: "b" },
            ],
          },
        ],
      };
      expect(HTTP.Task.deployReadConfigZ.validate(config)).toBe(false);
    });

    it("should not migrate a v0 header record on an endpoint", () => {
      // The Core migrates stored configs; the console only speaks generated shapes.
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            headers: { Accept: "application/json" },
          },
        ],
      };
      expect(HTTP.Task.READ_SCHEMAS.config.validate(config)).toBe(false);
    });

    it("should not migrate a v0 query param record on an endpoint", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            queryParams: { limit: "10" },
          },
        ],
      };
      expect(HTTP.Task.READ_SCHEMAS.config.validate(config)).toBe(false);
    });
  });

  describe("WRITE_SCHEMAS", () => {
    it("should validate the type literal", () => {
      expect(HTTP.Task.WRITE_SCHEMAS.type.parse(HTTP.Task.WRITE_TYPE)).toBe(
        HTTP.Task.WRITE_TYPE,
      );
    });

    it("should validate a config with a POST endpoint", () => {
      const config = {
        device: "dev-001",
        autoStart: false,
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api/control",
          },
        ],
      };
      const result = HTTP.Task.WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints).toHaveLength(1);
      expect(result.endpoints[0].method).toBe("POST");
    });

    it("should validate PUT and PATCH methods", () => {
      for (const method of ["PUT", "PATCH"] as const) {
        const config = {
          device: "dev-001",
          endpoints: [
            {
              key: "ep1",
              method,
              path: "/api",
            },
          ],
        };
        const result = HTTP.Task.WRITE_SCHEMAS.config.parse(config);
        expect(result.endpoints[0].method).toBe(method);
      }
    });

    it("should reject a read-only method at deploy", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            ...SENDS,
          },
        ],
      };
      expect(HTTP.Task.deployWriteConfigZ.validate(config)).toBe(false);
    });

    it("should default disabled to false", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
          },
        ],
      };
      const result = HTTP.Task.WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].disabled).toBe(false);
    });

    it("should validate endpoint with headers and query params", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            headers: [{ name: "X-Custom", value: "val" }],
            queryParams: [{ parameter: "key", value: "abc" }],
          },
        ],
      };
      const result = HTTP.Task.WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].headers).toEqual([{ name: "X-Custom", value: "val" }]);
      expect(result.endpoints[0].queryParams).toEqual([
        { parameter: "key", value: "abc" },
      ]);
    });

    it("should reject duplicate header names on a write endpoint at deploy", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            headers: [
              { name: "X-Key", value: "a" },
              { name: "X-Key", value: "b" },
            ],
            ...SENDS,
          },
        ],
      };
      expect(HTTP.Task.deployWriteConfigZ.validate(config)).toBe(false);
    });

    it("should reject duplicate query parameters on a write endpoint at deploy", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            queryParams: [
              { parameter: "key", value: "a" },
              { parameter: "key", value: "b" },
            ],
            ...SENDS,
          },
        ],
      };
      expect(HTTP.Task.deployWriteConfigZ.validate(config)).toBe(false);
    });

    it("should reject an enabled endpoint that sends no message at deploy", () => {
      const result = HTTP.Task.deployWriteConfigZ.safeParse({
        device: "dev-001",
        endpoints: [
          { key: "ep1", method: "POST", path: "/api", ...SENDS },
          { key: "ep2", method: "POST", path: "/off", disabled: true },
          { key: "ep3", method: "POST", path: "/none" },
        ],
      });
      expect(result.error?.issues).toMatchObject([
        { message: "A message is required", path: ["endpoints", 2, "message"] },
      ]);
    });
  });

  describe("write endpoint v0 shapes", () => {
    it("should not migrate a v0 header record on a write endpoint", () => {
      // The Core migrates stored configs; the console only speaks generated shapes.
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            headers: { "X-Custom": "val" },
          },
        ],
      };
      expect(HTTP.Task.WRITE_SCHEMAS.config.validate(config)).toBe(false);
    });

    it("should not migrate a v0 query param record on a write endpoint", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            queryParams: { key: "abc" },
          },
        ],
      };
      expect(HTTP.Task.WRITE_SCHEMAS.config.validate(config)).toBe(false);
    });
  });
});

describe("draft configs", () => {
  // Drafts persist server-side before configuration, so the shape schema must
  // accept every default config; retrieve parses with it.
  it("should accept the default read config", () => {
    const config = HTTP.Task.READ_SCHEMAS.config.parse({});
    expect(z.validate(HTTP.Task.READ_SCHEMAS.config, config)).toBe(true);
  });
  it("should accept the default write config", () => {
    const config = HTTP.Task.WRITE_SCHEMAS.config.parse({});
    expect(z.validate(HTTP.Task.WRITE_SCHEMAS.config, config)).toBe(true);
  });
});
