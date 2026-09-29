// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { test } from "vitest";
import { z } from "zod";

import { caseconv } from "@/caseconv";
import { record } from "@/record";

test("caseconv string conversion", async ({ bench }) => {
  const longSnake = Array.from({ length: 20 }, (_, i) => `segment_${i}`).join("_");
  const longCamel = Array.from({ length: 20 }, (_, i) => `Segment${i}`).join("");

  await bench.compare(
    bench("snakeToCamel - no conversion needed", () => {
      caseconv.snakeToCamel("alreadyCamelCase");
    }),

    bench("camelToSnake - no conversion needed", () => {
      caseconv.camelToSnake("already_snake_case");
    }),

    bench("snakeToCamel - short string", () => {
      caseconv.snakeToCamel("channel_key_name");
    }),

    bench("camelToSnake - short string", () => {
      caseconv.camelToSnake("channelKeyName");
    }),

    bench("snakeToCamel - long string (20 segments)", () => {
      caseconv.snakeToCamel(longSnake);
    }),

    bench("camelToSnake - long string (20 segments)", () => {
      caseconv.camelToSnake(longCamel);
    }),

    bench("snakeToCamel - OPC UA NodeId", () => {
      caseconv.snakeToCamel("ns=2;s=Temperature.Sensor.Value");
    }),

    bench("toKebab - from camelCase", () => {
      caseconv.toKebab("channelKeyName");
    }),

    bench("toKebab - from snake_case", () => {
      caseconv.toKebab("channel_key_name");
    }),

    bench("toProperNoun - from snake_case", () => {
      caseconv.toProperNoun("temperature_sensor_value");
    }),

    bench("toProperNoun - consecutive capitals", () => {
      caseconv.toProperNoun("XMLParserDocument");
    }),
  );
});

test("caseconv object conversion", async ({ bench }) => {
  const shallowObject = {
    channel_key: "test",
    data_type: "float32",
    is_index: false,
    sample_rate: 1000,
  };

  const nestedObject = {
    channel_config: {
      read_settings: { sample_rate: 1000, buffer_size: 4096 },
      write_settings: { enable_ack: true, timeout_ms: 5000 },
    },
  };

  const arrayOfObjects = Array.from({ length: 10 }, (_, i) => ({
    channel_key: `channel_${i}`,
    data_type: "float32",
    is_index: i === 0,
  }));

  const largeObject: Record<string, number> = {};
  for (let i = 0; i < 100; i++) largeObject[`property_name_${i}`] = i;

  const objectWithPrimitiveArrays = {
    channel_key: "test",
    data_values: [1.0, 2.0, 3.0, 4.0, 5.0],
    time_stamps: [100, 200, 300, 400, 500],
  };

  const objectWithUint8Array = {
    channel_key: "binary_data",
    raw_bytes: new Uint8Array([0x01, 0x02, 0x03, 0x04]),
  };

  await bench.compare(
    bench("snakeToCamel - shallow object (4 keys)", () => {
      caseconv.snakeToCamel(shallowObject);
    }),

    bench("snakeToCamel - nested object (3 levels)", () => {
      caseconv.snakeToCamel(nestedObject);
    }),

    bench("snakeToCamel - array of 10 objects", () => {
      caseconv.snakeToCamel(arrayOfObjects);
    }),

    bench("snakeToCamel - large object (100 keys)", () => {
      caseconv.snakeToCamel(largeObject);
    }),

    bench("snakeToCamel - object with primitive arrays", () => {
      caseconv.snakeToCamel(objectWithPrimitiveArrays);
    }),

    bench("snakeToCamel - object with Uint8Array", () => {
      caseconv.snakeToCamel(objectWithUint8Array);
    }),
  );
});

test("caseconv schema-based conversion", async ({ bench }) => {
  const schemaWithPreserve = z.object({
    task_config: z.object({
      channels: caseconv.preserveCase(z.record(z.string(), z.number())),
    }),
  });

  const inputWithOpcKeys = {
    task_config: {
      channels: {
        "ns=2;s=Temperature": 123,
        "ns=2;s=Pressure": 456,
        holding_register_input: 789,
      },
    },
  };

  const schemaWithoutPreserve = z.object({
    task_config: z.object({
      channels: z.record(z.string(), z.number()),
    }),
  });

  const complexSchema = z.object({
    read: z.object({
      index: z.number(),
      channels: caseconv.preserveCase(z.record(z.string(), z.number())),
    }),
    write: z.object({
      index: z.number(),
      channels: caseconv.preserveCase(z.record(z.string(), z.number())),
    }),
  });

  const complexInput = {
    read: { index: 0, channels: { "ns=2;s=Temp1": 1, "ns=2;s=Temp2": 2 } },
    write: { index: 1, channels: { "ns=2;s=Valve1": 3, "ns=2;s=Valve2": 4 } },
  };

  const deepSchema = z.object({
    level1: z.object({
      level2: z.object({
        level3: z.object({
          data: caseconv.preserveCase(z.record(z.string(), z.number())),
        }),
      }),
    }),
  });

  const deepInput = {
    level1: { level2: { level3: { data: { "ns=2;s=DeepValue": 42 } } } },
  };

  await bench.compare(
    bench("snakeToCamel - with preserveCase schema", () => {
      caseconv.snakeToCamel(inputWithOpcKeys, { schema: schemaWithPreserve });
    }),

    bench("snakeToCamel - same input without schema", () => {
      caseconv.snakeToCamel(inputWithOpcKeys);
    }),

    bench("snakeToCamel - schema without preserveCase", () => {
      caseconv.snakeToCamel(inputWithOpcKeys, { schema: schemaWithoutPreserve });
    }),

    bench("snakeToCamel - multiple preserveCase markers", () => {
      caseconv.snakeToCamel(complexInput, { schema: complexSchema });
    }),

    bench("snakeToCamel - deeply nested schema", () => {
      caseconv.snakeToCamel(deepInput, { schema: deepSchema });
    }),

    bench("camelToSnake - with preserveCase schema", () => {
      caseconv.camelToSnake(inputWithOpcKeys, { schema: schemaWithPreserve });
    }),
  );
});

test("caseconv wrapper traversal performance", async ({ bench }) => {
  const directPreserve = z.object({
    data: caseconv.preserveCase(z.record(z.string(), z.number())),
  });

  const optionalWrapped = z.object({
    data: caseconv.preserveCase(z.record(z.string(), z.number())).optional(),
  });

  const nullableWrapped = z.object({
    data: caseconv.preserveCase(z.record(z.string(), z.number())).nullable(),
  });

  const defaultWrapped = z.object({
    data: caseconv.preserveCase(z.record(z.string(), z.number())).default({}),
  });

  const transformWrapped = z.object({
    data: caseconv.preserveCase(z.record(z.string(), z.number())).transform((v) => v),
  });

  const deeplyNested = z.object({
    data: caseconv
      .preserveCase(z.record(z.string(), z.number()))
      .optional()
      .nullable()
      .default(null),
  });

  const unionSchema = z.object({
    data: record.nullishToEmpty(),
  });

  const dataInput = { data: { key_one: 1, key_two: 2 } };

  await bench.compare(
    bench("direct preserveCase (no traversal)", () => {
      caseconv.snakeToCamel(dataInput, { schema: directPreserve });
    }),

    bench("optional wrapped (1 level)", () => {
      caseconv.snakeToCamel(dataInput, { schema: optionalWrapped });
    }),

    bench("nullable wrapped (1 level)", () => {
      caseconv.snakeToCamel(dataInput, { schema: nullableWrapped });
    }),

    bench("default wrapped (1 level)", () => {
      caseconv.snakeToCamel(dataInput, { schema: defaultWrapped });
    }),

    bench("transform wrapped (1 level pipe)", () => {
      caseconv.snakeToCamel(dataInput, { schema: transformWrapped });
    }),

    bench("deeply nested (4 levels)", () => {
      caseconv.snakeToCamel(dataInput, { schema: deeplyNested });
    }),

    bench("union (nullishToEmpty)", () => {
      caseconv.snakeToCamel(dataInput, { schema: unionSchema });
    }),

    bench("no schema (baseline)", () => {
      caseconv.snakeToCamel(dataInput);
    }),
  );
});
