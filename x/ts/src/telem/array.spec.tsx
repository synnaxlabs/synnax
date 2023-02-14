// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { ValidationError } from "..";

import { LazyArray } from "./array";
import { DataType, TimeRange } from "./telem";

describe("LazyArray", () => {
  describe("construct", () => {
    test("valid from native", () => {
      const a = new LazyArray(new Float32Array([1, 2, 3]));
      expect(a.dataType.toString()).toBe(DataType.FLOAT32.toString());
      const b = new LazyArray(new BigInt64Array([BigInt(1)]));
      expect(b.dataType.toString()).toBe(DataType.INT64.toString());
      const c = new LazyArray(new BigInt64Array([BigInt(1)]), DataType.TIMESTAMP);
      expect(c.dataType.toString()).toBe(DataType.TIMESTAMP.toString());
    });

    test("from buffer without data type provided", () => {
      expect(() => {
        // eslint-disable-next-line no-new
        new LazyArray(new ArrayBuffer(4));
      }).toThrow(ValidationError);
    });

    test("from buffer with data type provided", () => {
      const a = new LazyArray(new ArrayBuffer(4), DataType.FLOAT32);
      expect(a.dataType.toString()).toBe(DataType.FLOAT32.toString());
    });

    test("with time range", () => {
      const a = new LazyArray(
        new Float32Array([1, 2, 3]),
        DataType.FLOAT32,
        new TimeRange(1, 2)
      );
      expect(a.timeRange.span.valueOf()).toBe(1);
    });
  });

  describe("convert", () => {
    test("from float64 to float32", () => {
      const a = new LazyArray(new Float64Array([1, 2, 3]), DataType.FLOAT64);
      const b = a.convert(DataType.FLOAT32);
      expect(b.dataType.toString()).toBe(DataType.FLOAT32.toString());
      expect(b.data).toEqual(new Float32Array([1, 2, 3]));
    });

    test("from int64 to int32", () => {
      const a = new LazyArray(new BigInt64Array([BigInt(1), BigInt(2), BigInt(3)]));
      const b = a.convert(DataType.INT32);
      expect(b.dataType.toString()).toBe(DataType.INT32.toString());
      expect(b.data).toEqual(new Int32Array([1, 2, 3]));
    });

    test("from float32 to int64", () => {
      const a = new LazyArray(new Float32Array([1, 2, 3]), DataType.FLOAT32);
      const b = a.convert(DataType.INT64);
      expect(b.dataType.toString()).toBe(DataType.INT64.toString());
      expect(b.data).toEqual(new BigInt64Array([BigInt(1), BigInt(2), BigInt(3)]));
    });
  });
});
