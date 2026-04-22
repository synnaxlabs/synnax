// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  READ_SCHEMAS,
  READ_TYPE,
  SCAN_SCHEMAS,
  WRITE_SCHEMAS,
  WRITE_TYPE,
} from "@/hardware/http/task/types";

describe("HTTP Task Types", () => {
  const readField = {
    pointer: "/value",
    channel: 1,
    enabled: true,
    key: "f1",
    dataType: DataType.FLOAT64.toString(),
  };

  describe("READ_SCHEMAS", () => {
    it("should validate the type literal", () => {
      expect(READ_SCHEMAS.type.parse(READ_TYPE)).toBe(READ_TYPE);
    });

    it("should validate a config with a GET endpoint", () => {
      const config = {
        device: "dev-001",
        dataSaving: true,
        autoStart: false,
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api/data",
            fields: [{ ...readField, pointer: "/temp" }],
          },
        ],
      };
      const result = READ_SCHEMAS.config.parse(config);
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
            fields: [readField],
          },
        ],
      };
      const result = READ_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].method).toBe("POST");
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
            fields: [readField],
          },
        ],
      };
      const result = READ_SCHEMAS.config.parse(config);
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
      const result = READ_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should reject a negative rate", () => {
      const config = {
        device: "dev-001",
        rate: -1,
        endpoints: [],
      };
      const result = READ_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should validate statusData as running/message object", () => {
      READ_SCHEMAS.statusData.parse({ running: true, message: "ok" });
    });

    it("should validate statusData as null", () => {
      READ_SCHEMAS.statusData.parse(null);
    });

    it("should validate statusData as undefined", () => {
      expect(READ_SCHEMAS.statusData.safeParse(undefined).success).toBe(true);
    });
  });

  describe("read field", () => {
    it("should validate a field with a valid JSON pointer", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            fields: [
              {
                pointer: "/data/temperature",
                channel: 1,
                enabled: true,
                key: "f1",
                dataType: DataType.FLOAT64.toString(),
              },
            ],
          },
        ],
      };
      const result = READ_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].fields[0].pointer).toBe("/data/temperature");
    });

    it("should reject an invalid JSON pointer", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            fields: [
              {
                pointer: "no-leading-slash",
                channel: 1,
                enabled: true,
                key: "f1",
              },
            ],
          },
        ],
      };
      const result = READ_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should validate a field with optional timestampFormat", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            fields: [
              {
                ...readField,
                pointer: "/ts",
                timestampFormat: "iso8601",
              },
            ],
          },
        ],
      };
      const result = READ_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].fields[0].timestampFormat).toBe("iso8601");
    });

    it("should reject an invalid timestampFormat", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            fields: [
              {
                pointer: "/ts",
                channel: 1,
                enabled: true,
                key: "f1",
                timestampFormat: "invalid_format",
              },
            ],
          },
        ],
      };
      const result = READ_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should validate a field with v1 enum values", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            fields: [
              {
                ...readField,
                pointer: "/status",
                enumValues: [
                  { label: "ON", value: 1 },
                  { label: "OFF", value: 0 },
                ],
              },
            ],
          },
        ],
      };
      const result = READ_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].fields[0].enumValues).toEqual([
        { label: "ON", value: 1 },
        { label: "OFF", value: 0 },
      ]);
    });

    it("should reject duplicate enum labels", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            fields: [
              {
                ...readField,
                pointer: "/status",
                enumValues: [
                  { label: "ON", value: 1 },
                  { label: "ON", value: 2 },
                ],
              },
            ],
          },
        ],
      };
      const result = READ_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });
  });

  describe("read enum v0 migration", () => {
    it("should migrate a v0 enum record to v1 array", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            fields: [
              {
                ...readField,
                pointer: "/status",
                enumValues: { ON: 1, OFF: 0 },
              },
            ],
          },
        ],
      };
      const result = READ_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].fields[0].enumValues).toEqual([
        { label: "ON", value: 1 },
        { label: "OFF", value: 0 },
      ]);
    });
  });

  describe("read endpoint", () => {
    it("should default index to null", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            fields: [],
          },
        ],
      };
      const result = READ_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].index).toBeNull();
    });

    it("should reject duplicate header names on an endpoint", () => {
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
            fields: [],
          },
        ],
      };
      const result = READ_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should reject duplicate query parameter names on an endpoint", () => {
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
            fields: [],
          },
        ],
      };
      const result = READ_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should migrate v0 header record on an endpoint", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            headers: { Accept: "application/json" },
            fields: [],
          },
        ],
      };
      const result = READ_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].headers).toEqual([
        { name: "Accept", value: "application/json" },
      ]);
    });

    it("should migrate v0 query param record on an endpoint", () => {
      const config = {
        device: "dev-001",
        rate: 1,
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            queryParams: { limit: "10" },
            fields: [],
          },
        ],
      };
      const result = READ_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].queryParams).toEqual([
        { parameter: "limit", value: "10" },
      ]);
    });
  });

  describe("WRITE_SCHEMAS", () => {
    it("should validate the type literal", () => {
      expect(WRITE_SCHEMAS.type.parse(WRITE_TYPE)).toBe(WRITE_TYPE);
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
            channel: { pointer: "/value", jsonType: "number", channel: 1 },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.parse(config);
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
              channel: { pointer: "/val", jsonType: "number", channel: 1 },
              fields: [],
            },
          ],
        };
        const result = WRITE_SCHEMAS.config.parse(config);
        expect(result.endpoints[0].method).toBe(method);
      }
    });

    it("should reject an invalid write method", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "GET",
            path: "/api",
            channel: { pointer: "/val", jsonType: "number", channel: 1 },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should default enabled to true", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "/val", jsonType: "number", channel: 1 },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].enabled).toBe(true);
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
            channel: { pointer: "/val", jsonType: "number", channel: 1 },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].headers).toEqual([{ name: "X-Custom", value: "val" }]);
      expect(result.endpoints[0].queryParams).toEqual([
        { parameter: "key", value: "abc" },
      ]);
    });

    it("should reject duplicate header names on a write endpoint", () => {
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
            channel: { pointer: "/val", jsonType: "number", channel: 1 },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should reject duplicate query parameter names on a write endpoint", () => {
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
            channel: { pointer: "/val", jsonType: "number", channel: 1 },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });
  });

  describe("write channel field", () => {
    it("should validate a channel field with all json types", () => {
      for (const jsonType of ["number", "string", "boolean"] as const) {
        const config = {
          device: "dev-001",
          endpoints: [
            {
              key: "ep1",
              method: "POST",
              path: "/api",
              channel: { pointer: "/val", jsonType, channel: 1 },
              fields: [],
            },
          ],
        };
        const result = WRITE_SCHEMAS.config.parse(config);
        expect(result.endpoints[0].channel.jsonType).toBe(jsonType);
      }
    });

    it("should reject an invalid jsonType", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "/val", jsonType: "object", channel: 1 },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should validate a channel field with enum values", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: {
              pointer: "/state",
              jsonType: "string",
              channel: 1,
              enumValues: [
                { value: 1, label: "ON" },
                { value: 0, label: "OFF" },
              ],
            },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].channel.enumValues).toHaveLength(2);
    });

    it("should reject duplicate write enum values", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: {
              pointer: "/state",
              jsonType: "string",
              channel: 1,
              enumValues: [
                { value: 1, label: "ON" },
                { value: 1, label: "OFF" },
              ],
            },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should validate a channel field with timeFormat", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: {
              pointer: "/ts",
              jsonType: "number",
              channel: 1,
              timeFormat: "unix_ms",
            },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].channel.timeFormat).toBe("unix_ms");
    });
  });

  describe("write endpoint custom checks", () => {
    it("should reject bare primitive pointer with additional fields", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "", jsonType: "number", channel: 1 },
            fields: [
              {
                key: "sf1",
                type: "static",
                pointer: "/extra",
                jsonType: "number",
                value: 42,
              },
            ],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should allow bare primitive pointer with no fields", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "", jsonType: "number", channel: 1 },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].channel.pointer).toBe("");
    });

    it("should reject duplicate pointers between channel and fields", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "/value", jsonType: "number", channel: 1 },
            fields: [
              {
                key: "sf1",
                type: "static",
                pointer: "/value",
                jsonType: "number",
                value: 42,
              },
            ],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should reject duplicate pointers between fields", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "/value", jsonType: "number", channel: 1 },
            fields: [
              {
                key: "sf1",
                type: "static",
                pointer: "/extra",
                jsonType: "number",
                value: 1,
              },
              {
                key: "sf2",
                type: "static",
                pointer: "/extra",
                jsonType: "number",
                value: 2,
              },
            ],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should reject a static field with empty pointer", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "/value", jsonType: "number", channel: 1 },
            fields: [
              {
                key: "sf1",
                type: "static",
                pointer: "",
                jsonType: "number",
                value: 42,
              },
            ],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should reject a generated field with empty pointer", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "/value", jsonType: "number", channel: 1 },
            fields: [{ key: "gf1", type: "generated", pointer: "", generator: "uuid" }],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });
  });

  describe("write fields", () => {
    it("should validate a static field", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "/value", jsonType: "number", channel: 1 },
            fields: [
              {
                key: "sf1",
                type: "static",
                pointer: "/device_id",
                jsonType: "string",
                value: "sensor-01",
              },
            ],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].fields[0].type).toBe("static");
    });

    it("should validate a generated UUID field", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "/value", jsonType: "number", channel: 1 },
            fields: [
              {
                key: "gf1",
                type: "generated",
                pointer: "/request_id",
                generator: "uuid",
              },
            ],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].fields[0].type).toBe("generated");
    });

    it("should validate a generated timestamp field with format", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "/value", jsonType: "number", channel: 1 },
            fields: [
              {
                key: "gf1",
                type: "generated",
                pointer: "/timestamp",
                generator: "timestamp",
                timeFormat: "iso8601",
              },
            ],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].fields[0].type).toBe("generated");
    });

    it("should reject an invalid generator type", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            channel: { pointer: "/value", jsonType: "number", channel: 1 },
            fields: [
              {
                key: "gf1",
                type: "generated",
                pointer: "/id",
                generator: "random",
              },
            ],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });
  });

  describe("write endpoint migrations", () => {
    it("should migrate v0 header record on a write endpoint", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            headers: { "X-Custom": "val" },
            channel: { pointer: "/val", jsonType: "number", channel: 1 },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].headers).toEqual([{ name: "X-Custom", value: "val" }]);
    });

    it("should migrate v0 query param record on a write endpoint", () => {
      const config = {
        device: "dev-001",
        endpoints: [
          {
            key: "ep1",
            method: "POST",
            path: "/api",
            queryParams: { key: "abc" },
            channel: { pointer: "/val", jsonType: "number", channel: 1 },
            fields: [],
          },
        ],
      };
      const result = WRITE_SCHEMAS.config.parse(config);
      expect(result.endpoints[0].queryParams).toEqual([
        { parameter: "key", value: "abc" },
      ]);
    });
  });
});

describe("HTTP Scan Task statusData", () => {
  it("should accept null", () => {
    expect(SCAN_SCHEMAS.statusData.safeParse(null).success).toBe(true);
  });
  it("should accept undefined", () => {
    expect(SCAN_SCHEMAS.statusData.safeParse(undefined).success).toBe(true);
  });
});
