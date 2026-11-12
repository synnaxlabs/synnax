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

import { http } from "@/http";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const READ_SCHEMAS = {
  type: z.literal("http_read"),
  config: http.readConfigZ,
  statusData: z.unknown(),
};

const parseField = (field: object) =>
  http.writeEndpointZ.parse({ fields: [{ key: "f", pointer: "/f", ...field }] })
    .fields[0];

describe("writeEndpointZ", () => {
  it("should keep a 64-bit integer field exact", () => {
    expect(parseField({ type: "integer", value: "-9223372036854775808" })).toEqual({
      key: "f",
      pointer: "/f",
      type: "integer",
      value: "-9223372036854775808",
    });
  });

  it("should default the time format of a timestamp field to ISO 8601", () => {
    expect(parseField({ type: "timestamp" })).toMatchObject({
      type: "timestamp",
      timeFormat: "iso8601",
    });
  });

  it("should read a non-finite float field", () => {
    expect(parseField({ type: "float", value: "NaN" })).toMatchObject({
      type: "float",
      value: NaN,
    });
  });

  it("should reject a field of an unknown type", () => {
    expect(() => parseField({ type: "static", value: 1 })).toThrow();
  });

  it("should reject a fixed field with no value", () => {
    expect(() => parseField({ type: "string" })).toThrow();
  });
});

describe("readConfigZ", () => {
  it("should keep a non-finite float fill and entry through a save", async () => {
    const config = http.readConfigZ.parse({
      rate: 1,
      endpoints: [
        {
          key: "ep",
          path: "/data",
          fields: [
            {
              type: "float",
              key: "f",
              pointer: "/v",
              fill: "NaN",
              enumValues: [{ label: "MAX", value: "Infinity" }],
            },
          ],
        },
      ],
    });
    const created = await client.tasks.create(
      { name: "http_read", type: "http_read", config },
      READ_SCHEMAS,
    );
    const retrieved = await client.tasks.retrieve({
      key: created.key,
      schemas: READ_SCHEMAS,
    });
    const field = retrieved.config.endpoints[0].fields[0];
    expect(field.type).toBe("float");
    if (field.type !== "float") return;
    expect(field.fill).toBeNaN();
    expect(field.enumValues[0].value).toBe(Infinity);
  });
});
