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

import { caseconv } from "@/caseconv";

describe("caseconv", () => {
  describe("snakeToCamel", () => {
    describe("strings", () => {
      const SPECS: [string, string][] = [
        ["foo_bar", "fooBar"],
        ["foo_bar_baz", "fooBarBaz"],
        ["foo", "foo"],
        ["fooBar", "fooBar"],
        ["fooBarBaz", "fooBarBaz"],
        ["NS=1;ID=5", "NS=1;ID=5"],
        ["foo-bar", "foo-bar"],
        ["FooBar", "fooBar"],
        ["foo.bar", "foo.bar"],
        ["foo_bar_baz.qux", "fooBarBaz.qux"],
      ];
      SPECS.forEach(([input, expected]) => {
        it(`should convert ${input} to ${expected}`, () => {
          expect(caseconv.snakeToCamel(input)).toBe(expected);
        });
      });
    });
    describe("objects", () => {
      const SPECS: [unknown, unknown][] = [
        [{ foo_bar: 1 }, { fooBar: 1 }],
        [{ foo_bar_baz: 1 }, { fooBarBaz: 1 }],
        [{ foo: 1 }, { foo: 1 }],
        [{ fooBar: 1 }, { fooBar: 1 }],
        [{ fooBarBaz: 1 }, { fooBarBaz: 1 }],
        [
          { NS: 1, ID: 5 },
          { NS: 1, ID: 5 },
        ],
        [{ "foo-bar": 1 }, { "foo-bar": 1 }],
        [{ foo_bar: { baz_qux: 1 } }, { fooBar: { bazQux: 1 } }],
        [[{ foo_bar: 1 }], [{ fooBar: 1 }]],
        [
          [{ foo_bar: 1 }, { baz_qux: 2 }],
          [{ fooBar: 1 }, { bazQux: 2 }],
        ],
        [
          { channel_key: "test", time_stamp: 123, value: [1, 2, 3] },
          { channelKey: "test", timeStamp: 123, value: [1, 2, 3] },
        ],
      ];
      SPECS.forEach(([input, expected], i) => {
        it(`should convert object ${i}`, () => {
          expect(caseconv.snakeToCamel(input)).toEqual(expected);
        });
      });
    });
  });
  describe("camelToSnake", () => {
    describe("strings", () => {
      const SPECS: [string, string][] = [
        ["fooBar", "foo_bar"],
        ["fooBarBaz", "foo_bar_baz"],
        ["foo", "foo"],
        ["foo_bar", "foo_bar"],
        ["foo_bar_baz", "foo_bar_baz"],
        ["NS=1;ID=5", "NS=1;ID=5"],
        ["foo-bar", "foo-bar"],
        ["foo.bar", "foo.bar"],
        ["fooBarBaz.qux", "foo_bar_baz.qux"],
      ];
      SPECS.forEach(([input, expected]) => {
        it(`should convert ${input} to ${expected}`, () => {
          expect(caseconv.camelToSnake(input)).toBe(expected);
        });
      });
    });
    describe("objects", () => {
      const SPECS: [unknown, unknown][] = [
        [{ fooBar: 1 }, { foo_bar: 1 }],
        [{ fooBarBaz: 1 }, { foo_bar_baz: 1 }],
        [{ foo: 1 }, { foo: 1 }],
        [{ foo_bar: 1 }, { foo_bar: 1 }],
        [{ foo_bar_baz: 1 }, { foo_bar_baz: 1 }],
        [
          { NS: 1, ID: 5 },
          { NS: 1, ID: 5 },
        ],
        [{ "foo-bar": 1 }, { "foo-bar": 1 }],
        [{ fooBar: { bazQux: 1 } }, { foo_bar: { baz_qux: 1 } }],
        [[{ fooBar: 1 }], [{ foo_bar: 1 }]],
        [
          [{ fooBar: 1 }, { bazQux: 2 }],
          [{ foo_bar: 1 }, { baz_qux: 2 }],
        ],
        [
          { channelKey: "test", timeStamp: 123, value: [1, 2, 3] },
          { channel_key: "test", time_stamp: 123, value: [1, 2, 3] },
        ],
      ];
      SPECS.forEach(([input, expected], i) => {
        it(`should convert object ${i}`, () => {
          expect(caseconv.camelToSnake(input)).toEqual(expected);
        });
      });
    });
  });
  describe("toKebab", () => {
    const SPECS: [string, string][] = [
      ["fooBar", "foo-bar"],
      ["fooBarBaz", "foo-bar-baz"],
      ["foo bar", "foo-bar"],
      ["foo bar baz", "foo-bar-baz"],
      ["foo.bar", "foo.bar"],
      ["foo.bar.baz", "foo.bar.baz"],
      ["Foo Bar", "foo-bar"],
      ["foo_bar", "foo-bar"],
      ["foo_bar_baz", "foo-bar-baz"],
    ];
    SPECS.forEach(([input, expected]) => {
      it(`should convert ${input} to ${expected}`, () => {
        expect(caseconv.toKebab(input)).toBe(expected);
      });
    });
  });
  describe("toProperNoun", () => {
    const SPECS: [string, string][] = [
      ["fooBar", "Foo Bar"],
      ["foo_bar", "Foo Bar"],
      ["foo-bar", "Foo Bar"],
      ["FooBar", "Foo Bar"],
      ["foo_bar_baz", "Foo Bar Baz"],
      ["fooBarBaz", "Foo Bar Baz"],
      ["foo-bar-baz", "Foo Bar Baz"],
      ["XMLParser", "XML Parser"],
      ["parseXMLDocument", "Parse XML Document"],
      ["IODevice", "IO Device"],
      ["temperature_sensor", "Temperature Sensor"],
      ["pressure-gauge", "Pressure Gauge"],
      ["flowMeter", "Flow Meter"],
      ["my_custom_symbol", "My Custom Symbol"],
      ["valve-actuator-v2", "Valve Actuator V2"],
      ["PIDController", "PID Controller"],
      ["", ""],
      ["a", "A"],
      ["ABC", "ABC"],
      ["test123value", "Test123value"],
      ["test_123_value", "Test 123 Value"],
    ];
    SPECS.forEach(([input, expected]) => {
      it(`should convert ${input} to ${expected}`, () => {
        expect(caseconv.toProperNoun(input)).toBe(expected);
      });
    });
  });

  describe("preserveCase", () => {
    describe("with ZodRecord", () => {
      it("should preserve case for record keys marked with preserveCase", () => {
        const schema = z.object({
          read: z.object({
            index: z.number(),
            channels: caseconv.preserveCase(z.record(z.string(), z.number())),
          }),
        });

        const input = {
          read: {
            index: 0,
            channels: {
              "ns=2;s=Temperature": 123,
              "i=2258": 456,
              holding_register_input: 789,
            },
          },
        };

        const result = caseconv.snakeToCamel(input, undefined, schema) as any;

        expect(result.read).toBeDefined();
        expect(result.read.index).toBe(0);
        expect(result.read.channels["ns=2;s=Temperature"]).toBe(123);
        expect(result.read.channels["i=2258"]).toBe(456);
        expect(result.read.channels.holding_register_input).toBe(789);
        expect(result.read.channels.holdingRegisterInput).toBeUndefined();
      });

      it("should preserve case for nested objects in marked records", () => {
        const schema = z.object({
          data: caseconv.preserveCase(
            z.record(z.string(), z.object({ value_name: z.number() })),
          ),
        });

        const input = {
          data: {
            "ns=2;s=Test": { value_name: 42 },
          },
        };

        const result = caseconv.snakeToCamel(input, undefined, schema) as any;

        expect(result.data["ns=2;s=Test"]).toBeDefined();
        expect(result.data["ns=2;s=Test"].value_name).toBe(42);
        expect(result.data["ns=2;s=Test"].valueName).toBeUndefined();
      });
    });

    describe("with ZodObject", () => {
      it("should preserve case for nested properties marked with preserveCase", () => {
        const schema = z.object({
          normalProp: z.string(),
          preservedProp: caseconv.preserveCase(
            z.object({
              nested_key: z.number(),
              another_nested: z.string(),
            }),
          ),
        });

        const input = {
          normal_prop: "test",
          preserved_prop: {
            nested_key: 123,
            another_nested: "value",
          },
        };

        const result = caseconv.snakeToCamel(input, undefined, schema) as any;

        expect(result.normalProp).toBe("test");
        expect(result.normal_prop).toBeUndefined();
        expect(result.preservedProp.nested_key).toBe(123);
        expect(result.preservedProp.another_nested).toBe("value");
        expect(result.preservedProp.nestedKey).toBeUndefined();
        expect(result.preservedProp.anotherNested).toBeUndefined();
      });
    });

    describe("without schema", () => {
      it("should convert normally when no schema is provided", () => {
        const input = {
          read: {
            channels: {
              "ns=2;s=Temperature": 123,
              holding_register_input: 456,
            },
          },
        };

        const result = caseconv.snakeToCamel(input) as any;

        expect(result.read.channels["ns=2;s=Temperature"]).toBe(123);
        expect(result.read.channels.holdingRegisterInput).toBe(456);
        expect(result.read.channels.holding_register_input).toBeUndefined();
      });
    });

    describe("camelToSnake with preserveCase", () => {
      it("should preserve case when encoding with camelToSnake", () => {
        const schema = z.object({
          read: z.object({
            channels: caseconv.preserveCase(z.record(z.string(), z.number())),
          }),
        });

        const input = {
          read: {
            channels: {
              "ns=2;s=Temperature": 123,
              "i=2258": 456,
            },
          },
        };

        const result = caseconv.camelToSnake(input, undefined, schema);

        // Record keys should NOT be converted
        expect(result.read.channels["ns=2;s=Temperature"]).toBe(123);
        expect(result.read.channels["i=2258"]).toBe(456);
      });
    });

    describe("comprehensive edge cases", () => {
      it("should handle empty preserved records", () => {
        const schema = z.object({
          channels: caseconv.preserveCase(z.record(z.string(), z.number())),
        });

        const result = caseconv.snakeToCamel({ channels: {} }, undefined, schema);
        expect(result.channels).toEqual({});
      });

      it("should handle arrays within preserved records", () => {
        const schema = z.object({
          channels: caseconv.preserveCase(z.record(z.string(), z.array(z.number()))),
        });

        const input = {
          channels: {
            "ns=2;s=ArrayChannel": [1, 2, 3],
          },
        };

        const result = caseconv.snakeToCamel(input, undefined, schema) as any;
        expect(result.channels["ns=2;s=ArrayChannel"]).toEqual([1, 2, 3]);
      });

      it("should handle deeply nested preserved objects", () => {
        const schema = z.object({
          outer: z.object({
            inner: caseconv.preserveCase(
              z.object({
                deep_value: z.number(),
              }),
            ),
          }),
        });

        const input = {
          outer: {
            inner: {
              deep_value: 42,
            },
          },
        };

        const result = caseconv.snakeToCamel(input, undefined, schema) as any;
        expect(result.outer.inner.deep_value).toBe(42);
        expect(result.outer.inner.deepValue).toBeUndefined();
      });

      it("should handle multiple preserved records at different levels", () => {
        const schema = z.object({
          read: z.object({
            channels: caseconv.preserveCase(z.record(z.string(), z.number())),
          }),
          write: z.object({
            channels: caseconv.preserveCase(z.record(z.string(), z.number())),
          }),
        });

        const input = {
          read: {
            channels: {
              holding_register_input: 123,
            },
          },
          write: {
            channels: {
              coil_output: 456,
            },
          },
        };

        const result = caseconv.snakeToCamel(input, undefined, schema) as any;
        expect(result.read.channels.holding_register_input).toBe(123);
        expect(result.write.channels.coil_output).toBe(456);
      });

      it("should handle Modbus channel keys with hyphens and underscores", () => {
        const schema = z.object({
          channels: caseconv.preserveCase(z.record(z.string(), z.number())),
        });

        const input = {
          channels: {
            "holding_register_input-100-float32": 123,
            "register_input-200": 456,
            "coil_input-5": 789,
          },
        };

        const result = caseconv.snakeToCamel(input, undefined, schema) as any;
        expect(result.channels["holding_register_input-100-float32"]).toBe(123);
        expect(result.channels["register_input-200"]).toBe(456);
        expect(result.channels["coil_input-5"]).toBe(789);
      });
    });
  });
});
