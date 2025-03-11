// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { schema } from "@/schema";

interface TestCase<T extends number | bigint> {
  value: T;
  success: boolean;
}

const test = <T extends number | bigint>(
  schema: z.ZodSchema<T>,
  cases: TestCase<T>[],
) => {
  cases.forEach(({ value, success }) => {
    if (success)
      it(`should parse ${value}`, () => {
        expect(schema.parse(value)).toBe(value);
      });
    else
      it(`should reject ${value}`, () => {
        expect(() => schema.parse(value)).toThrow();
      });
  });
};

describe("Integer Schemas", () => {
  describe("int8", () => {
    const testCases: TestCase<z.infer<typeof schema.int8>>[] = [
      { value: -Infinity, success: false },
      { value: -129, success: false },
      { value: -128, success: true },
      { value: -1, success: true },
      { value: 0, success: true },
      { value: 1, success: true },
      { value: 1.5, success: false },
      { value: 127, success: true },
      { value: 128, success: false },
      { value: Infinity, success: false },
      { value: NaN, success: false },
    ];
    test(schema.int8, testCases);
  });

  describe("int16", () => {
    const testCases: TestCase<z.infer<typeof schema.int16>>[] = [
      { value: -Infinity, success: false },
      { value: -32769, success: false },
      { value: -32768, success: true },
      { value: -1, success: true },
      { value: 0, success: true },
      { value: 1, success: true },
      { value: 1.5, success: false },
      { value: 32767, success: true },
      { value: 32768, success: false },
      { value: Infinity, success: false },
      { value: NaN, success: false },
    ];
    test(schema.int16, testCases);
  });

  describe("int32", () => {
    const testCases: TestCase<z.infer<typeof schema.int32>>[] = [
      { value: -Infinity, success: false },
      { value: -2147483649, success: false },
      { value: -2147483648, success: true },
      { value: -1, success: true },
      { value: 0, success: true },
      { value: 1, success: true },
      { value: 1.5, success: false },
      { value: 2147483647, success: true },
      { value: 2147483648, success: false },
      { value: Infinity, success: false },
      { value: NaN, success: false },
    ];
    test(schema.int32, testCases);
  });

  describe("int64", () => {
    const testCases: TestCase<z.infer<typeof schema.int64>>[] = [
      { value: -9223372036854775809n, success: false },
      { value: -9223372036854775808n, success: true },
      { value: -1n, success: true },
      { value: 0n, success: true },
      { value: 1n, success: true },
      { value: 9223372036854775807n, success: true },
      { value: 9223372036854775808n, success: false },
    ];
    test(schema.int64, testCases);
  });

  describe("uint8", () => {
    const testCases: TestCase<z.infer<typeof schema.uint8>>[] = [
      { value: -Infinity, success: false },
      { value: -1, success: false },
      { value: 0, success: true },
      { value: 1, success: true },
      { value: 1.5, success: false },
      { value: 255, success: true },
      { value: 256, success: false },
      { value: Infinity, success: false },
      { value: NaN, success: false },
    ];
    test(schema.uint8, testCases);
  });

  describe("uint16", () => {
    const testCases: TestCase<z.infer<typeof schema.uint16>>[] = [
      { value: -Infinity, success: false },
      { value: -1, success: false },
      { value: 0, success: true },
      { value: 1, success: true },
      { value: 1.5, success: false },
      { value: 65535, success: true },
      { value: 65536, success: false },
      { value: Infinity, success: false },
      { value: NaN, success: false },
    ];
    test(schema.uint16, testCases);
  });

  describe("uint32", () => {
    const testCases: TestCase<z.infer<typeof schema.uint32>>[] = [
      { value: -Infinity, success: false },
      { value: -1, success: false },
      { value: 0, success: true },
      { value: 1, success: true },
      { value: 1.5, success: false },
      { value: 4294967295, success: true },
      { value: 4294967296, success: false },
      { value: Infinity, success: false },
      { value: NaN, success: false },
    ];
    test(schema.uint32, testCases);
  });

  describe("uint64", () => {
    const testCases: TestCase<z.infer<typeof schema.uint64>>[] = [
      { value: -1n, success: false },
      { value: 0n, success: true },
      { value: 1n, success: true },
      { value: 18446744073709551615n, success: true },
      { value: 18446744073709551616n, success: false },
    ];
    test(schema.uint64, testCases);
  });
});
