// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";
import { z } from "zod/v4";

import { MockGLBufferController } from "@/mock/MockGLBufferController";
import { type CrudeSeries, isCrudeSeries, MultiSeries, Series } from "@/telem/series";
import {
  type CrudeDataType,
  DataType,
  Size,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@/telem/telem";

describe("Series", () => {
  describe("construction", () => {
    const IS_CRUDE_SERIES_SPEC: Array<[unknown, boolean]> = [
      [{}, false],
      [{ constructor: {} }, false],
      [1, true],
      [[1, 2, 3], true],
      [["a", "b", "c"], true],
      [new Float32Array([12]), true],
    ];
    IS_CRUDE_SERIES_SPEC.forEach(([value, expected]) => {
      it(`should return ${expected} for ${JSON.stringify(value)}`, () => {
        expect(isCrudeSeries(value)).toEqual(expected);
      });
    });

    describe("length", () => {
      it("should return the correct length for a fixed density series", () => {
        const a = new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
        });
        expect(a.length).toEqual(3);
      });
      it("should return the correct length for a variable density series", () => {
        const a = new Series({
          data: [{ value: 1 }, { value: 2, red: "blue" }, { value: 3, dog: "14" }],
        });
        expect(a.dataType.equals(DataType.JSON)).toBeTruthy();
        expect(a.length).toEqual(3);
        const buf = a.data.buffer;
        const c = new Series({ data: buf, dataType: DataType.JSON });
        expect(c.length).toEqual(3);
      });
    });

    test("from another series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: new TimeRange(TimeStamp.seconds(5), TimeStamp.seconds(20)),
      });
      const b = new Series(a);
      expect(b.buffer).toBe(a.buffer);
      expect(b.timeRange).toBe(a.timeRange);
    });

    test("valid from typed array", () => {
      const a = new Series(new Float32Array([1, 2, 3]));
      expect(a.dataType.toString()).toBe(DataType.FLOAT32.toString());
      expect(a.length).toEqual(3);
      expect(a.byteLength).toEqual(Size.bytes(12));
      expect(a.byteCapacity).toEqual(Size.bytes(12));
      expect(a.capacity).toEqual(3);
      const b = new Series({ data: new BigInt64Array([BigInt(1)]) });
      expect(b.dataType.toString()).toBe(DataType.INT64.toString());
      const c = new Series({
        data: new BigInt64Array([BigInt(1)]),
        dataType: DataType.TIMESTAMP,
      });
      expect(c.dataType.toString()).toBe(DataType.TIMESTAMP.toString());
    });

    test("from buffer without data type provided", () => {
      expect(() => {
        new Series({ data: new ArrayBuffer(4) });
      }).toThrow();
    });

    test("valid from js array", () => {
      const a = new Series({ data: [1, 2, 3], dataType: DataType.FLOAT32 });
      expect(a.dataType.toString()).toBe(DataType.FLOAT32.toString());
      expect(a.length).toEqual(3);
      expect(a.byteLength).toEqual(Size.bytes(12));
      expect(a.byteCapacity).toEqual(Size.bytes(12));
      expect(a.capacity).toEqual(3);
    });

    it("should assume float64 when JS numbers are passed as data", () => {
      const s = new Series({ data: [1, 2, 3] });
      expect(s.dataType.equals(DataType.FLOAT64));
    });

    it("should assume float64 when a single number is passed in", () => {
      const s = new Series(1);
      expect(s.dataType.equals(DataType.FLOAT64)).toBeTruthy();
    });

    it("should assume string when JS strings are passed as data", () => {
      const s = new Series({ data: ["apple", "banana", "carrot"] });
      expect(s.dataType.equals(DataType.STRING));
      expect(s.length).toEqual(3);
    });

    it("should construct a series from an int32 array", () => {
      const s = new Series(new Int32Array([1, 2, 3]));
      expect(s.dataType.equals(DataType.INT32)).toBeTruthy();
      expect(s.length).toEqual(3);
      expect(Array.from(s)).toEqual([1, 2, 3]);
    });

    it("should assume string when a single string is passed as data", () => {
      const s = new Series("abc");
      expect(s.dataType.equals(DataType.STRING)).toBeTruthy();
      expect(s.length).toEqual(1);
    });

    it("should assume JSON when JS objects are passed as data", () => {
      const s = new Series({ data: [{ a: 1, b: "apple" }] });
      expect(s.dataType.equals(DataType.JSON));
      expect(s.length).toEqual(1);
      expect(s.data.at(-1)).toEqual(10);
    });

    it("should correctly interpret a bigint as an int64", () => {
      const s = new Series(1n);
      expect(s.dataType.equals(DataType.INT64)).toBeTruthy();
      expect(s.length).toEqual(1);
    });

    it("should correctly interpret a TimeStamp object as a data type of timestamp", () => {
      const s = new Series(TimeStamp.now());
      expect(s.dataType).toEqual(DataType.TIMESTAMP);
    });

    it("should correctly interpret an array of TimeStamps as a data type of timestamp", () => {
      const s = new Series([TimeStamp.now(), TimeStamp.now().add(TimeSpan.seconds(1))]);
      expect(s.dataType).toEqual(DataType.TIMESTAMP);
    });

    it("should correctly interpret a Date object as a data type of timestamp", () => {
      const s = new Series(new Date());
      expect(s.dataType).toEqual(DataType.TIMESTAMP);
    });

    it("should encode objects as JSON", () => {
      const a = new Series({ data: [{ a: 1, b: "apple" }], dataType: DataType.JSON });
      expect(a.dataType.toString()).toBe(DataType.JSON.toString());
    });

    it("should convert a numeric value to a BigInt when data type is int64, timestamp, or uint64", () => {
      const a = new Series({ data: 12, dataType: DataType.INT64 });
      expect(a.dataType.toString()).toBe(DataType.INT64.toString());
      expect(a.data).toEqual(new BigInt64Array([BigInt(12)]));
      const b = new Series({ data: 12, dataType: DataType.TIMESTAMP });
      expect(b.dataType.toString()).toBe(DataType.TIMESTAMP.toString());
      expect(b.data).toEqual(new BigInt64Array([BigInt(12)]));
      const c = new Series({ data: 12, dataType: DataType.UINT64 });
      expect(c.dataType.toString()).toBe(DataType.UINT64.toString());
      expect(c.data).toEqual(new BigUint64Array([BigInt(12)]));
    });

    it("should convert an array of numeric values to a BigInt when data type is int64, timestamp, or uint64", () => {
      const a = new Series({ data: [12, 13, 14], dataType: DataType.INT64 });
      expect(a.dataType.toString()).toBe(DataType.INT64.toString());
      expect(a.data).toEqual(new BigInt64Array([BigInt(12), BigInt(13), BigInt(14)]));
      const b = new Series({ data: [12, 13, 14], dataType: DataType.TIMESTAMP });
      expect(b.dataType.toString()).toBe(DataType.TIMESTAMP.toString());
      expect(b.data).toEqual(new BigInt64Array([BigInt(12), BigInt(13), BigInt(14)]));
      const c = new Series({ data: [12, 13, 14], dataType: DataType.UINT64 });
      expect(c.dataType.toString()).toBe(DataType.UINT64.toString());
      expect(c.data).toEqual(new BigUint64Array([BigInt(12), BigInt(13), BigInt(14)]));
    });

    it("should convert bigints to numbers when data type does not use bigints", () => {
      const a = new Series({ data: [12n, 13n, 14n], dataType: DataType.FLOAT32 });
      expect(a.dataType.toString()).toBe(DataType.FLOAT32.toString());
      expect(a.data).toEqual(new Float32Array([12, 13, 14]));
    });

    it("should correctly convert a mix of bigints and numbers", () => {
      const a = new Series({ data: [12n, 13], dataType: DataType.FLOAT32 });
      expect(a.dataType.toString()).toBe(DataType.FLOAT32.toString());
      expect(a.data).toEqual(new Float32Array([12, 13]));
    });

    it("should convert a floating point numeric value to a BigInt when data type is int64, timestamp, or uint64", () => {
      const a = new Series({ data: 12.5, dataType: DataType.INT64 });
      expect(a.dataType.toString()).toBe(DataType.INT64.toString());
      expect(a.data).toEqual(new BigInt64Array([BigInt(13)]));
    });

    it("should convert encoded keys to snake_case", () => {
      const a = new Series({ data: [{ aB: 1, bC: "apple" }], dataType: DataType.JSON });
      const strContent = new TextDecoder().decode(a.data);
      expect(strContent).toBe('{"a_b":1,"b_c":"apple"}\n');
    });

    it("should throw an error when an empty JS array is provided and no data type is provided", () => {
      expect(() => {
        new Series({ data: [] });
      }).toThrow(
        "cannot infer data type from a zero length JS array when constructing a Series. Please provide a data type.",
      );
    });

    it("should throw an error when constructing a series from a symbol", () => {
      expect(() => {
        new Series({ data: [Symbol("123")] });
      }).toThrow(
        "cannot infer data type of symbol when constructing a Series from a JS array",
      );
    });

    it("should decode keys from snake_case to camelCase", () => {
      const a = new Series({
        data: [{ a_b: 1, b_c: "apple" }],
        dataType: DataType.JSON,
      });
      const obj = a.at(0);
      expect(obj).toEqual({ aB: 1, bC: "apple" });
    });

    it("should correctly separate strings", () => {
      const a = new Series({ data: ["apple"], dataType: DataType.STRING });
      expect(a.dataType.toString()).toBe(DataType.STRING.toString());
    });

    test("from buffer with data type provided", () => {
      const a = new Series({ data: new ArrayBuffer(4), dataType: DataType.FLOAT32 });
      expect(a.dataType.toString()).toBe(DataType.FLOAT32.toString());
    });

    test("with time range", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
        timeRange: new TimeRange(1, 2),
      });
      expect(a.timeRange.span.valueOf()).toBe(1n);
    });

    describe("allocation", () => {
      it("should allocate a lazy array", () => {
        const series = Series.alloc({ capacity: 10, dataType: DataType.FLOAT32 });
        expect(series.byteCapacity).toEqual(Size.bytes(40));
        expect(series.capacity).toEqual(10);
        expect(series.length).toEqual(0);
        expect(series.byteLength).toEqual(Size.bytes(0));
      });
      it("should throw an error when attempting to allocate an array of lenght 0", () => {
        expect(() => {
          Series.alloc({ capacity: 0, dataType: DataType.FLOAT32 });
        }).toThrow();
      });
    });
  });

  describe("at", () => {
    it("should return the value at the given index and add the sample offset", () => {
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
        sampleOffset: 2,
      });
      expect(series.at(0)).toEqual(3);
      expect(series.at(1)).toEqual(4);
      expect(series.at(2)).toEqual(5);
    });
    it("should return undefined when the index is out of bounds", () => {
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      expect(series.at(3)).toBeUndefined();
    });
    it("should allow the index to be negative", () => {
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      expect(series.at(-1)).toEqual(3);
    });
    it("should throw an error when the index is out of bounds and require is set to true", () => {
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      expect(() => {
        series.at(3, true);
      }).toThrow();
    });
    it("should return the correct value for a string series", () => {
      const series = new Series({
        data: ["apple", "banana", "carrot"],
        dataType: DataType.STRING,
      });
      expect(series.at(0)).toEqual("apple");
      expect(series.at(1)).toEqual("banana");
      expect(series.at(2)).toEqual("carrot");
      expect(series.at(-1)).toEqual("carrot");
    });
    it("should return the correct value for a JSON series", () => {
      const series = new Series({
        data: [
          { a: 1, b: "apple" },
          { a: 2, b: "banana" },
        ],
        dataType: DataType.JSON,
      });
      expect(series.at(0)).toEqual({ a: 1, b: "apple" });
      expect(series.at(1)).toEqual({ a: 2, b: "banana" });
    });
    it("should throw an error if the index is out of bounds for a string series", () => {
      const series = new Series({
        data: ["apple", "banana", "carrot"],
        dataType: DataType.STRING,
      });
      expect(() => {
        series.at(3, true);
      }).toThrow();
    });
    it("should throw an error if the index is out of bounds for a JSON series", () => {
      const series = new Series({
        data: [
          { a: 1, b: "apple" },
          { a: 2, b: "banana" },
        ],
        dataType: DataType.JSON,
      });
      expect(() => {
        series.at(2, true);
      }).toThrow();
    });
  });

  describe("atAlignment", () => {
    it("should return the value at a particular alignment", () => {
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
        alignment: 1n,
      });
      expect(series.atAlignment(1n)).toEqual(1);
      expect(series.atAlignment(2n)).toEqual(2);
      expect(series.atAlignment(3n)).toEqual(3);
    });
  });

  describe("slice", () => {
    it("should slice a lazy array", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const b = a.slice(1, 2);
      expect(b.dataType.toString()).toBe(DataType.FLOAT32.toString());
      expect(b.data).toEqual(new Float32Array([2]));
      expect(b.length).toEqual(1);
      expect(b.byteLength).toEqual(Size.bytes(4));
      expect(b.byteCapacity).toEqual(Size.bytes(4));
      expect(b.capacity).toEqual(1);
    });
  });

  describe("min and max", () => {
    describe("numbers", () => {
      it("should return a min and max of zero on an allocated array", () => {
        const series = Series.alloc({ capacity: 10, dataType: DataType.FLOAT32 });
        expect(series.max).toEqual(-Infinity);
        expect(series.min).toEqual(Infinity);
      });
      it("should correctly calculate the min and max of a lazy array", () => {
        const series = new Series({
          data: new Float32Array([1, 2, 3]),
          dataType: DataType.FLOAT32,
        });
        expect(series.max).toEqual(3);
        expect(series.min).toEqual(1);
      });
    });
    describe("bigints", () => {
      it("should return a min and max of zero on an allocated array", () => {
        const series = Series.alloc({ capacity: 10, dataType: DataType.INT64 });
        expect(series.max).toEqual(-Infinity);
        expect(series.min).toEqual(Infinity);
      });
      it("should correctly calculate the min and max of a lazy array", () => {
        const series = new Series({
          data: new BigInt64Array([1n, 2n, 3n]),
          dataType: DataType.INT64,
        });
        expect(series.max).toEqual(3n);
        expect(series.min).toEqual(1n);
      });
    });
    it("should throw an error if that data type is not numeric", () => {
      const series = new Series({
        data: ["a", "b", "c"],
        dataType: DataType.STRING,
      });
      expect(() => series.min).toThrow(
        "cannot calculate minimum on a variable length data type",
      );
      expect(() => series.max).toThrow(
        "cannot calculate maximum on a variable length data type",
      );
    });
  });

  describe("conversion", () => {
    test("from float64 to float32", () => {
      const a = new Series({
        data: new Float64Array([1, 2, 3]),
        dataType: DataType.FLOAT64,
      });
      const b = a.convert(DataType.FLOAT32);
      expect(b.dataType.toString()).toBe(DataType.FLOAT32.toString());
      expect(b.data).toEqual(new Float32Array([1, 2, 3]));
    });

    test("from int64 to int32", () => {
      const a = new Series({
        data: new BigInt64Array([BigInt(1), BigInt(2), BigInt(3)]),
      });
      const b = a.convert(DataType.INT32);
      expect(b.dataType.toString()).toBe(DataType.INT32.toString());
      expect(b.data).toEqual(new Int32Array([1, 2, 3]));
    });

    test("from float32 to int64", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const b = a.convert(DataType.INT64);
      expect(b.dataType.toString()).toBe(DataType.INT64.toString());
      expect(b.data).toEqual(new BigInt64Array([BigInt(1), BigInt(2), BigInt(3)]));
    });
  });

  describe("write", () => {
    it("should correctly write to an allocated lazy array", () => {
      const series = Series.alloc({ capacity: 10, dataType: DataType.FLOAT32 });
      expect(series.byteCapacity).toEqual(Size.bytes(40));
      expect(series.length).toEqual(0);
      const writeOne = new Series({ data: new Float32Array([1]) });
      expect(series.write(writeOne)).toEqual(1);
      expect(series.length).toEqual(1);
      const writeTwo = new Series({ data: new Float32Array([2, 3]) });
      expect(series.write(writeTwo)).toEqual(2);
      expect(series.length).toEqual(3);
    });

    it("should recompute cached max and min correctly", () => {
      const series = Series.alloc({ capacity: 10, dataType: DataType.FLOAT32 });
      expect(series.max).toEqual(-Infinity);
      expect(series.min).toEqual(Infinity);
      const writeOne = new Series({ data: new Float32Array([2, 3]) });
      series.write(writeOne);
      expect(series.max).toEqual(3);
      expect(series.min).toEqual(2);
      const writeTwo = new Series({ data: new Float32Array([4, 5]) });
      series.write(writeTwo);
      expect(series.max).toEqual(5);
      expect(series.min).toEqual(2);
    });

    it("should recompute the length of a variable density array", () => {
      const series = Series.alloc({ capacity: 12, dataType: DataType.STRING });
      expect(series.length).toEqual(0);
      const writeOne = new Series({ data: ["apple"] });
      expect(series.write(writeOne)).toEqual(1);
      expect(series.length).toEqual(1);
      const writeTwo = new Series({ data: ["apple"] });
      expect(series.write(writeTwo)).toEqual(1);
      expect(series.length).toEqual(2);
    });

    it("should correctly adjust the sample offset of a written array", () => {
      const series = Series.alloc({
        capacity: 2,
        dataType: DataType.FLOAT32,
        timeRange: TimeRange.ZERO,
        sampleOffset: -3,
      });
      const writeOne = new Series({ data: new Float32Array([-2]) });
      expect(series.write(writeOne)).toEqual(1);
      expect(series.min).toEqual(-5);
      const writeTwo = new Series({
        data: new Float32Array([1]),
        dataType: DataType.FLOAT32,
        timeRange: TimeRange.ZERO,
        sampleOffset: -1,
      });
      expect(series.write(writeTwo)).toEqual(1);
      expect(series.min).toEqual(-5);
      expect(series.max).toEqual(-2);
    });
  });

  describe("webgl buffering", () => {
    it("should correctly buffer a new lazy array", () => {
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const controller = new MockGLBufferController();
      series.updateGLBuffer(controller);
      expect(controller.createBufferMock).toHaveBeenCalledTimes(1);
      expect(controller.bindBufferMock).toHaveBeenCalledTimes(1);
      expect(controller.bufferDataMock).toHaveBeenCalledTimes(1);
      expect(controller.bufferSubDataMock).not.toHaveBeenCalled();
      const buf = controller.buffers[series.glBuffer as number];
      expect(buf).toBeDefined();
      expect(buf.byteLength).toEqual(12);
      expect(buf).toEqual(new Float32Array([1, 2, 3]).buffer);
    });

    it("should correctly update a buffer when writing to an allocated array", () => {
      const series = Series.alloc({ capacity: 10, dataType: DataType.FLOAT32 });
      const controller = new MockGLBufferController();
      series.updateGLBuffer(controller);
      expect(controller.createBufferMock).toHaveBeenCalledTimes(1);
      expect(controller.bindBufferMock).not.toHaveBeenCalled();
      expect(controller.bufferDataMock).not.toHaveBeenCalled();
      expect(controller.bufferSubDataMock).not.toHaveBeenCalled();
      let buf = controller.buffers[series.glBuffer as number];
      expect(buf).toBeDefined();
      expect(buf.byteLength).toEqual(0);
      const writeOne = new Series({ data: new Float32Array([1]) });
      series.write(writeOne);
      series.updateGLBuffer(controller);
      expect(controller.bufferDataMock).toHaveBeenCalledTimes(1);
      expect(controller.bufferSubDataMock).toHaveBeenCalledTimes(1);
      buf = controller.buffers[series.glBuffer as number];
      expect(buf.byteLength).toEqual(series.byteCapacity.valueOf());
      expect(new Float32Array(buf)[0]).toEqual(1);
      const writeTwo = new Series({ data: new Float32Array([2, 3]) });
      series.write(writeTwo);
      series.updateGLBuffer(controller);
      expect(controller.bufferDataMock).not.toHaveBeenCalledTimes(2);
      expect(controller.bufferSubDataMock).toHaveBeenCalledTimes(2);
      buf = controller.buffers[series.glBuffer as number];
      expect(buf.byteLength).toEqual(series.byteCapacity.valueOf());
      expect(new Float32Array(buf)[0]).toEqual(1);
      expect(new Float32Array(buf)[1]).toEqual(2);
      expect(new Float32Array(buf)[2]).toEqual(3);
    });
    it("should correctly de-allocate the buffer when the reference counter drops to 0", () => {
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const controller = new MockGLBufferController();
      series.acquire(controller);
      expect(controller.createBufferMock).toHaveBeenCalledTimes(1);
      series.release();
      expect(controller.deleteBufferMock).toHaveBeenCalledTimes(1);
      expect(() => series.glBuffer).toThrow();
      expect(Object.keys(controller.buffers)).toHaveLength(0);
    });
    it("should allocate the buffer again when the reference counter goes back up from 0", () => {
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const controller = new MockGLBufferController();
      series.acquire(controller);
      series.release();
      series.acquire(controller);
      expect(controller.createBufferMock).toHaveBeenCalledTimes(2);
      const buf = controller.buffers[series.glBuffer as number];
      expect(buf.byteLength).toEqual(series.byteCapacity.valueOf());
    });
  });

  describe("acquire", () => {
    it("should increase the reference count and buffer gl data", () => {
      const s = new Series({ data: new Float32Array([1, 2, 3]) });
      expect(s.refCount).toEqual(0);
      const control = new MockGLBufferController();
      s.acquire(control);
      expect(s.refCount).toEqual(1);
      expect(control.createBufferMock).toHaveBeenCalled();
      s.release();
      expect(s.refCount).toEqual(0);
      expect(control.deleteBufferMock).toHaveBeenCalled();
    });
  });

  describe("string series", () => {
    it("should correctly encode and decode a string series", () => {
      const s = new Series(["apple", "banana", "carrot"]);
      expect(s.dataType.toString()).toEqual(DataType.STRING.toString());
      const outStrings = s.toStrings();
      expect(outStrings).toEqual(["apple", "banana", "carrot"]);
    });

    it("should allow allocation of a particular byte capacity", () => {
      const s = Series.alloc({ capacity: 10, dataType: DataType.STRING });
      expect(s.byteCapacity).toEqual(Size.bytes(10));
    });

    it("should allow a caller to write to the series", () => {
      const s = Series.alloc({ capacity: 10, dataType: DataType.STRING });
      const writeOne = new Series({ data: ["apple"] });
      const written = s.write(writeOne);
      expect(written).toEqual(1);
      expect(s.length).toEqual(1);
      expect(s.at(0)).toEqual("apple");
    });

    it("should allow a caller to write to the series multiple times", () => {
      const s = Series.alloc({ capacity: 100, dataType: DataType.STRING });
      const writeOne = new Series({ data: ["apple"] });
      const writeTwo = new Series({ data: ["banana", "carrot"] });
      const written = s.write(writeOne);
      expect(written).toEqual(1);
      const writtenTwo = s.write(writeTwo);
      expect(writtenTwo).toEqual(2);
      expect(s.length).toEqual(3);
      expect(s.at(0)).toEqual("apple");
      expect(s.at(1)).toEqual("banana");
      expect(s.at(2)).toEqual("carrot");
    });

    it("should prevent the caller from writing past the series capacity", () => {
      const s = Series.alloc({ capacity: 10, dataType: DataType.STRING });
      const writeOne = new Series({ data: ["apple"] });
      const writeTwo = new Series({ data: ["banana", "carrot"] });
      const written = s.write(writeOne);
      expect(written).toEqual(1);
      const writtenTwo = s.write(writeTwo);
      expect(writtenTwo).toEqual(0);
    });
  });

  describe("JSON series", () => {
    it("should correctly encode and decode a JSON series", () => {
      const schema = z.object({
        a: z.number(),
        b: z.string(),
      });
      const s = new Series([
        { a: 1, b: "apple" },
        { a: 2, b: "banana" },
        { a: 3, b: "carrot" },
      ]);
      const outJSON = s.parseJSON(schema);
      expect(outJSON).toEqual([
        { a: 1, b: "apple" },
        { a: 2, b: "banana" },
        { a: 3, b: "carrot" },
      ]);
    });
  });

  describe("binarySearch", () => {
    it("should correctly binary search a pre-allocated array", () => {
      const series = Series.alloc({ capacity: 10, dataType: DataType.FLOAT32 });
      const writeOne = new Series({ data: new Float32Array([1, 2, 3, 4, 5]) });
      series.write(writeOne);
      expect(series.binarySearch(3)).toEqual(2);
      expect(series.binarySearch(6)).toEqual(5);
    });
  });

  describe("array construction", () => {
    it("should correctly a JS array from a series", () => {
      const s = new Series([1, 2, 3]);
      const arr = Array.from(s);
      expect(arr).toEqual([1, 2, 3]);
    });
    it("should correctly construct a js array from a string series", () => {
      const s = new Series(["apple", "banana", "carrot"]);
      const arr = Array.from(s);
      expect(arr).toEqual(["apple", "banana", "carrot"]);
    });
    it("should correctly construct a JS array from a JSON series", () => {
      const s = new Series([
        { a: 1, b: "apple" },
        { a: 2, b: "banana" },
        { a: 3, b: "carrot" },
      ]);
      const arr = Array.from(s);
      expect(arr).toEqual([
        { a: 1, b: "apple" },
        { a: 2, b: "banana" },
        { a: 3, b: "carrot" },
      ]);
    });
  });

  describe("as", () => {
    describe("number", () => {
      it("should correctly interpret the series as numeric", () => {
        const s = new Series([1, 2, 3]);
        const s2 = s.as("number");
        expect(s2.at(0)).toEqual(1);
      });
      it("should throw an error if the series is not numeric", () => {
        const s = new Series(["a", "b", "c"]);
        expect(() => {
          s.as("number");
        }).toThrow();
      });
    });
    describe("string", () => {
      it("should correctly interpret the series as a string", () => {
        const s = new Series(["apple", "banana", "carrot"]);
        const s2 = s.as("string");
        expect(s2.at(0)).toEqual("apple");
      });
      it("should throw an error if the series is not a string", () => {
        const s = new Series([1, 2, 3]);
        expect(() => {
          s.as("string");
        }).toThrow();
      });
    });
    describe("bigint", () => {
      it("should correctly interpret the series as a bigint", () => {
        const s = new Series([BigInt(1), BigInt(2), BigInt(3)]);
        const s2 = s.as("bigint");
        expect(s2.at(0)).toEqual(BigInt(1));
      });
      it("should throw an error if the series is not a bigint", () => {
        const s = new Series([1, 2, 3]);
        expect(() => {
          s.as("bigint");
        }).toThrow();
      });
    });
  });

  describe("alignmentBounds", () => {
    it("should correctly return the alignment bounds of a multi-series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: new TimeRange(1, 2),
        alignment: 1n,
      });
      expect(a.alignmentBounds).toEqual({ lower: 1n, upper: 4n });
    });
  });

  describe("toStrings", () => {
    interface Spec {
      name: string;
      values: CrudeSeries;
      expected: string[];
      dataType?: CrudeDataType;
    }
    const SPECS: Spec[] = [
      {
        name: "string",
        values: ["apple", "banana", "carrot"],
        expected: ["apple", "banana", "carrot"],
      },
      {
        name: "json",
        values: [
          { a: 1, b: "apple" },
          { a: 2, b: "banana" },
          { a: 3, b: "carrot" },
        ],
        expected: [
          '{"a":1,"b":"apple"}',
          '{"a":2,"b":"banana"}',
          '{"a":3,"b":"carrot"}',
        ],
        dataType: "json",
      },
      {
        name: "number",
        values: [1, 2, 3],
        expected: ["1", "2", "3"],
      },
      {
        name: "bigint",
        values: [BigInt(1), BigInt(2), BigInt(3)],
        expected: ["1", "2", "3"],
      },
    ];
    SPECS.forEach(({ name, values, expected }) => {
      it(`should correctly convert a ${name} series to strings`, () => {
        const s = new Series({ data: values });
        expect(s.toStrings()).toEqual(expected);
      });
    });

    it("should return an series of length 0 if the series is empty", () => {
      const s = new Series({ data: new Float32Array([]), dataType: DataType.STRING });
      const outStrings = s.toStrings();
      expect(outStrings).toEqual([]);
    });
  });

  describe("toUUIDs", () => {
    it("should convert a UUID series to an array of UUID strings", () => {
      // Valid UUID v4 bytes (version 4, variant 1)
      const bytes = new Uint8Array([
        // First UUID: 123e4567-e89b-4xxx-yxxx-426614174000 (version 4, variant 1)
        0x12, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x40, 0xd3, 0x80, 0x56, 0x42, 0x66, 0x14,
        0x17, 0x40, 0x00,
        // Second UUID: 7f3e4567-e89b-4xxx-yxxx-426614174000 (version 4, variant 1)
        0x7f, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x40, 0xd3, 0x80, 0x56, 0x42, 0x66, 0x14,
        0x17, 0x40, 0x00,
      ]);
      const series = new Series({ data: bytes, dataType: DataType.UUID });
      const uuids = series.toUUIDs();
      expect(uuids).toHaveLength(2);
      expect(uuids[0]).toBe("123e4567-e89b-40d3-8056-426614174000");
      expect(uuids[1]).toBe("7f3e4567-e89b-40d3-8056-426614174000");
    });

    it("should throw an error when converting non-UUID series", () => {
      const series = new Series({ data: [1, 2, 3], dataType: DataType.INT32 });
      expect(() => series.toUUIDs()).toThrow("cannot convert non-uuid series to uuids");
    });

    it("should handle empty UUID series", () => {
      const series = new Series({ data: new Uint8Array(), dataType: DataType.UUID });
      const uuids = series.toUUIDs();
      expect(uuids).toHaveLength(0);
    });

    it("should handle series with nil UUID", () => {
      // Nil UUID: 00000000-0000-0000-0000-000000000000
      const bytes = new Uint8Array(16).fill(0);
      const series = new Series({ data: bytes, dataType: DataType.UUID });
      const uuids = series.toUUIDs();
      expect(uuids).toHaveLength(1);
      expect(uuids[0]).toBe("00000000-0000-0000-0000-000000000000");
    });

    it("should handle series with max UUID", () => {
      // Max UUID: ffffffff-ffff-ffff-ffff-ffffffffffff
      const bytes = new Uint8Array(16).fill(0xff);
      const series = new Series({ data: bytes, dataType: DataType.UUID });
      const uuids = series.toUUIDs();
      expect(uuids).toHaveLength(1);
      expect(uuids[0]).toBe("ffffffff-ffff-ffff-ffff-ffffffffffff");
    });
  });

  describe("digest", () => {
    it("should return a digest of information about the series", () => {
      const digest = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: new TimeRange(1, 3),
      }).digest;
      expect(digest.alignment.lower).toEqual({ domain: 0n, sample: 0n });
      expect(digest.alignment.upper).toEqual({ domain: 0n, sample: 3n });
      expect(digest.dataType).toEqual("float32");
      expect(digest.length).toEqual(3);
      expect(digest.timeRange).toEqual(new TimeRange(1, 3).toString());
    });
  });

  describe("parse", () => {
    it("should correctly parse a minimum series", () => {
      const s = Series.z.parse({ dataType: "uint8" });
      expect(s.dataType.equals(DataType.UINT8)).toBeTruthy();
      expect(s.length).toEqual(0);
    });
    it("should correctly parse a string buffer for data", () => {
      const s = Series.z.parse({ data: "", dataType: "string" });
      expect(s.dataType.equals(DataType.STRING)).toBeTruthy();
      expect(s.length).toEqual(0);
    });
    it("should correctly parse a series with null data", () => {
      const s = Series.z.parse({ data: null, dataType: "string" });
      expect(s.dataType.equals(DataType.STRING)).toBeTruthy();
      expect(s.length).toEqual(0);
    });
  });

  describe("toString", () => {
    interface Spec {
      series: Series;
      expected: string;
    }
    const SPECS: Spec[] = [
      {
        series: new Series({ data: [1, 2, 3, 4], dataType: "float64" }),
        expected: "Series(float64 4 [1,2,3,4])",
      },
      {
        series: new Series({
          data: Array.from({ length: 100 }, (_, i) => i),
          dataType: "float32",
        }),
        expected: "Series(float32 100 [0,1,2,3,4,...,95,96,97,98,99])",
      },
    ];
    SPECS.forEach(({ series, expected }) => {
      it(`should convert ${series.toString()} to a string`, () => {
        expect(series.toString()).toEqual(expected);
      });
    });
  });

  describe("sub", () => {
    it("should return a sub-series backed by the same buffer", () => {
      const arr = new Float32Array([1, 2, 3, 4, 5]);
      const v2 = arr.subarray(1, 4);
      expect(v2.buffer).toBe(arr.buffer);
      const s1 = new Series(arr);
      expect(s1.buffer).toBe(arr.buffer);
    });
  });

  describe("subIter", () => {
    it("should return an iterator over a sub-series", () => {
      const s = new Series(new Float32Array([1, 2, 3, 4, 5]));
      const iter = s.subIterator(1, 4);
      expect(iter.next().value).toEqual(2);
      expect(iter.next().value).toEqual(3);
      expect(iter.next().value).toEqual(4);
    });
  });

  describe("subIterAlignment", () => {
    it("should return an iterator over a sub-series", () => {
      const s = new Series({
        data: new Float32Array([1, 2, 3, 4, 5]),
        alignment: 2n,
      });
      const iter = s.subAlignmentIterator(3n, 5n);
      expect(iter.next().value).toEqual(2);
      expect(iter.next().value).toEqual(3);
      expect(iter.next().done).toBeTruthy();
    });
    it("should clamp the bounds to the alignment", () => {
      const s = new Series({
        data: new Float32Array([1, 2, 3, 4, 5]),
        alignment: 2n,
      });
      const iter = s.subAlignmentIterator(1n, 5n);
      expect(iter.next().value).toEqual(1);
      expect(iter.next().value).toEqual(2);
      expect(iter.next().value).toEqual(3);
      expect(iter.next().done).toBeTruthy();
    });
  });

  describe("bounds", () => {
    it("should return the bounds of the series", () => {
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      expect(series.bounds).toEqual({ lower: 1, upper: 3 });
    });

    it("should handle negative numbers", () => {
      const series = new Series({
        data: new Float32Array([-3, -2, -1]),
        dataType: DataType.FLOAT32,
      });
      expect(series.bounds).toEqual({ lower: -3, upper: -1 });
    });

    it("should handle empty series", () => {
      const series = new Series({
        data: new Float32Array([]),
        dataType: DataType.FLOAT32,
      });
      expect(series.bounds).toEqual({ lower: Infinity, upper: -Infinity });
    });
  });

  describe("reAlign", () => {
    it("should create a new series with the specified alignment", () => {
      const original = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
        alignment: 0n,
      });
      const realigned = original.reAlign(10n);
      expect(realigned.alignment).toBe(10n);
      expect(realigned.alignmentBounds).toEqual({ lower: 10n, upper: 13n });
      expect(realigned.data).toEqual(original.data);
    });

    it("should preserve data when realigning", () => {
      const original = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
        alignment: 0n,
      });
      const realigned = original.reAlign(10n);
      expect(realigned.at(0)).toBe(1);
      expect(realigned.at(1)).toBe(2);
      expect(realigned.at(2)).toBe(3);
    });

    it("should update alignment bounds correctly", () => {
      const original = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
        alignment: 0n,
      });
      const realigned = original.reAlign(10n);
      expect(realigned.alignmentBounds).toEqual({ lower: 10n, upper: 13n });
    });

    it("should handle atAlignment with new alignment", () => {
      const original = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
        alignment: 0n,
      });
      const realigned = original.reAlign(10n);
      expect(realigned.atAlignment(10n)).toBe(1);
      expect(realigned.atAlignment(11n)).toBe(2);
      expect(realigned.atAlignment(12n)).toBe(3);
    });

    it("should handle bigint series", () => {
      const original = new Series({
        data: new BigInt64Array([1n, 2n, 3n]),
        dataType: DataType.INT64,
        alignment: 0n,
      });
      const realigned = original.reAlign(10n);
      expect(realigned.alignment).toBe(10n);
      expect(realigned.alignmentBounds).toEqual({ lower: 10n, upper: 13n });
      expect(realigned.at(0)).toBe(1n);
      expect(realigned.at(1)).toBe(2n);
      expect(realigned.at(2)).toBe(3n);
    });

    it("should handle variable length data types", () => {
      const original = new Series({
        data: ["a", "b", "c"],
        dataType: DataType.STRING,
        alignment: 0n,
      });
      const realigned = original.reAlign(10n);
      expect(realigned.alignment).toBe(10n);
      expect(realigned.alignmentBounds).toEqual({ lower: 10n, upper: 13n });
      expect(realigned.at(0)).toBe("a");
      expect(realigned.at(1)).toBe("b");
      expect(realigned.at(2)).toBe("c");
    });

    it("should preserve sample offset", () => {
      const original = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
        alignment: 0n,
        sampleOffset: 10,
      });
      const realigned = original.reAlign(10n);
      expect(realigned.sampleOffset).toBe(10);
      expect(realigned.at(0)).toBe(11);
      expect(realigned.at(1)).toBe(12);
      expect(realigned.at(2)).toBe(13);
    });
  });
});

describe("MultiSeries", () => {
  describe("length", () => {
    it("should correctly return the length of the multi-series", () => {
      const a = new Series(new Float32Array([1, 2, 3]));
      const b = new Series(new Float32Array([4, 5, 6]));
      const multi = new MultiSeries([a, b]);
      expect(multi.length).toEqual(6);
    });
  });

  describe("at", () => {
    it("should correctly return the value at a particular index", () => {
      const a = new Series(new Float32Array([1, 2, 3]));
      const b = new Series(new Float32Array([4, 5, 6]));
      const multi = new MultiSeries([a, b]);
      expect(multi.at(0)).toEqual(1);
      expect(multi.at(1)).toEqual(2);
      expect(multi.at(2)).toEqual(3);
      expect(multi.at(3)).toEqual(4);
      expect(multi.at(4)).toEqual(5);
      expect(multi.at(5)).toEqual(6);
    });
    it("should correctly return a value via negative indexing", () => {
      const a = new Series(new Float32Array([1, 2, 3]));
      const b = new Series(new Float32Array([4, 5, 6]));
      const multi = new MultiSeries([a, b]);
      expect(multi.at(-1)).toEqual(6);
      expect(multi.at(-2)).toEqual(5);
      expect(multi.at(-3)).toEqual(4);
      expect(multi.at(-4)).toEqual(3);
      expect(multi.at(-5)).toEqual(2);
      expect(multi.at(-6)).toEqual(1);
    });

    it("should return undefined if the index is not found", () => {
      const a = new Series(new Float32Array([1, 2, 3]));
      const b = new Series(new Float32Array([4, 5, 6]));
      const multi = new MultiSeries([a, b]);
      expect(multi.at(10)).toBeUndefined();
    });

    it("should throw an error if the index is not found and required is true", () => {
      const a = new Series(new Float32Array([1, 2, 3]));
      const b = new Series(new Float32Array([4, 5, 6]));
      const multi = new MultiSeries([a, b]);
      expect(() => multi.at(10, true)).toThrow();
    });
  });

  describe("atAlignment", () => {
    it("should correctly return the value at a particular alignment", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        alignment: 1n,
      });
      const b = new Series({
        data: new Float32Array([4, 5, 6]),
        alignment: 5n,
      });
      const multi = new MultiSeries([a, b]);
      expect(multi.atAlignment(1n)).toEqual(1);
      expect(multi.atAlignment(2n)).toEqual(2);
      expect(multi.atAlignment(3n)).toEqual(3);
      expect(multi.atAlignment(5n)).toEqual(4);
      expect(multi.atAlignment(6n)).toEqual(5);
      expect(multi.atAlignment(7n)).toEqual(6);
    });

    it("should return undefined if the alignment is not found", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        alignment: 1n,
      });
      const multi = new MultiSeries([a]);
      expect(multi.atAlignment(45n)).toBeUndefined();
    });

    it("should throw if the alignment is not found and required is true", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        alignment: 1n,
      });
      const multi = new MultiSeries([a]);
      expect(() => multi.atAlignment(45n, true)).toThrow();
    });
  });

  describe("subIterator", () => {
    it("should return an iterator over a sub-series", () => {
      const a = new Series(new Float32Array([1, 2, 3, 4, 5]));
      const b = new Series(new Float32Array([6, 7, 8, 9, 10]));
      const multi = new MultiSeries([a, b]);
      const iter = multi.subIterator(1, 8);
      expect(iter.next().value).toEqual(2);
      expect(iter.next().value).toEqual(3);
      expect(iter.next().value).toEqual(4);
      expect(iter.next().value).toEqual(5);
      expect(iter.next().value).toEqual(6);
      expect(iter.next().value).toEqual(7);
      expect(iter.next().value).toEqual(8);
      expect(iter.next().done).toBeTruthy();
    });
  });

  describe("subAlignmentIterator", () => {
    it("should return an iterator over a sub-series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3, 4, 5]),
        alignment: 2n,
      });
      const b = new Series({
        data: new Float32Array([6, 7, 8, 9, 10]),
        alignment: 8n,
      });
      const multi = new MultiSeries([a, b]);
      const iter = multi.subAlignmentIterator(3n, 9n);
      expect(iter.next().value).toEqual(2);
      expect(iter.next().value).toEqual(3);
      expect(iter.next().value).toEqual(4);
      expect(iter.next().value).toEqual(5);
      expect(iter.next().value).toEqual(6);
      expect(iter.next().done).toBeTruthy();
    });

    it("Should work correctly when starting at an alignment before the first series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3, 4, 5]),
        alignment: 2n,
      });
      const b = new Series({
        data: new Float32Array([6, 7, 8, 9, 10]),
        alignment: 8n,
      });
      const multi = new MultiSeries([a, b]);
      const iter = multi.subAlignmentIterator(1n, 9n);
      expect(iter.next().value).toEqual(1);
      expect(iter.next().value).toEqual(2);
      expect(iter.next().value).toEqual(3);
      expect(iter.next().value).toEqual(4);
      expect(iter.next().value).toEqual(5);
      expect(iter.next().value).toEqual(6);
      expect(iter.next().done).toBeTruthy();
    });

    it("should work correctly when staring at an alignment equal to the upper bound of the first series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3, 4, 5]),
        alignment: 2n,
      });
      const b = new Series({
        data: new Float32Array([6, 7, 8, 9, 10]),
        alignment: 8n,
      });
      const multi = new MultiSeries([a, b]);
      const iter = multi.subAlignmentIterator(7n, 10n);
      expect(iter.next().value).toEqual(6);
      expect(iter.next().value).toEqual(7);
      expect(iter.next().done).toBeTruthy();
    });

    it("should work correctly when the starting alignment is between two series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3, 4, 5]),
        alignment: 2n,
      });
      const b = new Series({
        data: new Float32Array([6, 7, 8, 9, 10]),
        alignment: 10n,
      });
      const multi = new MultiSeries([a, b]);
      const iter = multi.subAlignmentIterator(7n, 12n);
      expect(iter.next().value).toEqual(6);
      expect(iter.next().value).toEqual(7);
      expect(iter.next().done).toBeTruthy();
    });

    it("Should work correctly when ending at an alignment after the last series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3, 4, 5]),
        alignment: 2n,
      });
      const b = new Series({
        data: new Float32Array([6, 7, 8, 9, 10]),
        alignment: 8n,
      });
      const multi = new MultiSeries([a, b]);
      const iter = multi.subAlignmentIterator(3n, 20n);
      expect(iter.next().value).toEqual(2);
      expect(iter.next().value).toEqual(3);
      expect(iter.next().value).toEqual(4);
      expect(iter.next().value).toEqual(5);
      expect(iter.next().value).toEqual(6);
      expect(iter.next().value).toEqual(7);
      expect(iter.next().value).toEqual(8);
      expect(iter.next().value).toEqual(9);
      expect(iter.next().value).toEqual(10);
      expect(iter.next().done).toBeTruthy();
    });
  });

  describe("parseJSON", () => {
    it("should correctly parse a multi-series of JSON", () => {
      const a = new Series([
        { a: 1, b: "apple" },
        { a: 2, b: "banana" },
      ]);
      const b = new Series([
        { a: 3, b: "carrot" },
        { a: 4, b: "dog" },
      ]);
      const multi = new MultiSeries([a, b]);
      const arr = multi.parseJSON(z.object({ a: z.number(), b: z.string() }));
      expect(arr).toEqual([
        { a: 1, b: "apple" },
        { a: 2, b: "banana" },
        { a: 3, b: "carrot" },
        { a: 4, b: "dog" },
      ]);
    });
  });

  describe("array construction", () => {
    it("should correctly construct a JS array from a multi-series", () => {
      const a = new Series(new Float32Array([1, 2, 3]));
      const b = new Series(new Float32Array([4, 5, 6]));
      const multi = new MultiSeries([a, b]);
      const arr = Array.from(multi);
      expect(arr).toEqual([1, 2, 3, 4, 5, 6]);
    });
    it("should correctly construct a JS array from a multi-series of strings", () => {
      const a = new Series(["apple", "banana", "carrot"]);
      const b = new Series(["dog", "elephant", "fox"]);
      const multi = new MultiSeries([a, b]);
      const arr = Array.from(multi);
      expect(arr).toEqual(["apple", "banana", "carrot", "dog", "elephant", "fox"]);
    });
    it("should correctly construct a JS array from a multi-series of JSON", () => {
      const a = new Series([
        { a: 1, b: "apple" },
        { a: 2, b: "banana" },
        { a: 3, b: "carrot" },
      ]);
      const b = new Series([
        { a: 4, b: "dog" },
        { a: 5, b: "elephant" },
        { a: 6, b: "fox" },
      ]);
      const multi = new MultiSeries([a, b]);
      const arr = Array.from(multi);
      expect(arr).toEqual([
        { a: 1, b: "apple" },
        { a: 2, b: "banana" },
        { a: 3, b: "carrot" },
        { a: 4, b: "dog" },
        { a: 5, b: "elephant" },
        { a: 6, b: "fox" },
      ]);
    });
    it("should correctly construct a JS array from a multi-series with no series", () => {
      const multi = new MultiSeries([]);
      const arr = Array.from(multi);
      expect(arr).toEqual([]);
    });
  });

  describe("data", () => {
    it("should combine the data of the series into a single typed array", () => {
      const a = new Series(new Float32Array([1, 2, 3]));
      const b = new Series(new Float32Array([4, 5, 6]));
      const multi = new MultiSeries([a, b]);
      const data = multi.data;
      expect(data).toEqual(new Float32Array([1, 2, 3, 4, 5, 6]));
    });
  });

  describe("timeRange", () => {
    it("should correctly return the time range of a multi-series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: new TimeRange(1, 2),
      });
      const b = new Series({
        data: new Float32Array([4, 5, 6]),
        timeRange: new TimeRange(3, 4),
      });
      const multi = new MultiSeries([a, b]);
      expect(multi.timeRange).toEqual(new TimeRange(1, 4));
    });
  });

  describe("as", () => {
    it("should correctly cast a numeric series to number type", () => {
      const a = new Series(new Float32Array([1, 2, 3]));
      const b = new Series(new Float32Array([4, 5, 6]));
      const multi = new MultiSeries([a, b]);
      const asNum = multi.as("number");
      expect(asNum.at(0)).toEqual(1);
      expect(asNum.at(5)).toEqual(6);
      expect(Array.from(asNum)).toEqual([1, 2, 3, 4, 5, 6]);
    });

    it("should correctly cast a string series to string type", () => {
      const a = new Series(["apple", "banana"]);
      const b = new Series(["carrot", "date"]);
      const multi = new MultiSeries([a, b]);
      const asStr = multi.as("string");
      expect(asStr.at(0)).toEqual("apple");
      expect(asStr.at(3)).toEqual("date");
      expect(Array.from(asStr)).toEqual(["apple", "banana", "carrot", "date"]);
    });

    it("should correctly cast a bigint series to bigint type", () => {
      const a = new Series([1n, 2n]);
      const b = new Series([3n, 4n]);
      const multi = new MultiSeries([a, b]);
      const asBigInt = multi.as("bigint");
      expect(asBigInt.at(0)).toEqual(1n);
      expect(asBigInt.at(3)).toEqual(4n);
      expect(Array.from(asBigInt)).toEqual([1n, 2n, 3n, 4n]);
    });

    it("should throw an error when trying to cast to an incompatible type", () => {
      const a = new Series(new Float32Array([1, 2, 3]));
      const b = new Series(new Float32Array([4, 5, 6]));
      const multi = new MultiSeries([a, b]);
      expect(() => multi.as("string")).toThrow();
    });
  });

  describe("bounds", () => {
    it("should return bounds of [Infinity, -Infinity] for an empty MultiSeries", () => {
      const multiSeries = new MultiSeries();
      expect(multiSeries.bounds).toEqual({ lower: Infinity, upper: -Infinity });
    });

    it("should correctly calculate bounds across multiple series", () => {
      const series1 = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const series2 = new Series({
        data: new Float32Array([4, 5, 6]),
        dataType: DataType.FLOAT32,
      });
      const multiSeries = new MultiSeries([series1, series2]);
      expect(multiSeries.bounds).toEqual({ lower: 1, upper: 6 });
    });

    it("should handle negative numbers across series", () => {
      const series1 = new Series({
        data: new Float32Array([-5, -3, -1]),
        dataType: DataType.FLOAT32,
      });
      const series2 = new Series({
        data: new Float32Array([0, 2, 4]),
        dataType: DataType.FLOAT32,
      });
      const multiSeries = new MultiSeries([series1, series2]);
      expect(multiSeries.bounds).toEqual({ lower: -5, upper: 4 });
    });

    it("should handle sample offsets across series", () => {
      const series1 = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
        sampleOffset: 10,
      });
      const series2 = new Series({
        data: new Float32Array([4, 5, 6]),
        dataType: DataType.FLOAT32,
        sampleOffset: 20,
      });
      const multiSeries = new MultiSeries([series1, series2]);
      expect(multiSeries.bounds).toEqual({ lower: 11, upper: 26 });
    });

    it("should handle bigint series", () => {
      const series1 = new Series({
        data: new BigInt64Array([1n, 2n, 3n]),
        dataType: DataType.INT64,
      });
      const series2 = new Series({
        data: new BigInt64Array([4n, 5n, 6n]),
        dataType: DataType.INT64,
      });
      const multiSeries = new MultiSeries([series1, series2]);
      expect(multiSeries.bounds).toEqual({ lower: 1, upper: 6 });
    });

    it("should throw an error for non-numeric data types", () => {
      const series1 = new Series({
        data: ["a", "b", "c"],
        dataType: DataType.STRING,
      });
      const series2 = new Series({
        data: ["d", "e", "f"],
        dataType: DataType.STRING,
      });
      const multiSeries = new MultiSeries([series1, series2]);
      expect(() => multiSeries.bounds).toThrow(
        "cannot calculate minimum on a variable length data type",
      );
    });
  });

  describe("subAlignmentSpanIterator", () => {
    it("should return an empty iterator when start is beyond bounds", () => {
      const s1 = new Series({ data: [1, 2, 3], alignment: 0n });
      const s2 = new Series({ data: [4, 5, 6], alignment: 3n });
      const ms = new MultiSeries([s1, s2]);
      const iter = ms.subAlignmentSpanIterator(10n, 5);
      expect(Array.from(iter)).toEqual([]);
    });

    it("should return an empty iterator when start is at upper bound", () => {
      const s1 = new Series({ data: [1, 2, 3], alignment: 0n });
      const s2 = new Series({ data: [4, 5, 6], alignment: 3n });
      const ms = new MultiSeries([s1, s2]);
      const iter = ms.subAlignmentSpanIterator(6n, 5);
      expect(Array.from(iter)).toEqual([]);
    });

    it("should iterate over samples within a single series", () => {
      const s1 = new Series({ data: [1, 2, 3], alignment: 0n });
      const s2 = new Series({ data: [4, 5, 6], alignment: 3n });
      const ms = new MultiSeries([s1, s2]);
      const iter = ms.subAlignmentSpanIterator(1n, 2);
      expect(Array.from(iter)).toEqual([2, 3]);
    });

    it("should iterate over samples across multiple series", () => {
      const s1 = new Series({ data: [1, 2, 3], alignment: 0n });
      const s2 = new Series({ data: [4, 5, 6], alignment: 3n });
      const ms = new MultiSeries([s1, s2]);
      const iter = ms.subAlignmentSpanIterator(2n, 3);
      expect(Array.from(iter)).toEqual([3, 4, 5]);
    });

    it("should handle span that exceeds available data", () => {
      const s1 = new Series({ data: [1, 2, 3], alignment: 0n });
      const s2 = new Series({ data: [4, 5, 6], alignment: 3n });
      const ms = new MultiSeries([s1, s2]);
      const iter = ms.subAlignmentSpanIterator(4n, 10);
      expect(Array.from(iter)).toEqual([5, 6]);
    });

    it("should handle span that exceeds available data with non-continuous spans", () => {
      const s1 = new Series({ data: [1, 2, 3], alignment: 0n });
      const s2 = new Series({ data: [4, 5, 6], alignment: 500000000n });
      const ms = new MultiSeries([s1, s2]);
      const iter = ms.subAlignmentSpanIterator(4n, 10000000000);
      expect(Array.from(iter)).toEqual([4, 5, 6]);
    });

    it("should handle empty series", () => {
      const ms = new MultiSeries();
      const iter = ms.subAlignmentSpanIterator(0n, 5);
      expect(Array.from(iter)).toEqual([]);
    });

    it("should handle start alignment before first series", () => {
      const s1 = new Series({ data: [1, 2, 3], alignment: 2n });
      const s2 = new Series({ data: [4, 5, 6], alignment: 5n });
      const ms = new MultiSeries([s1, s2]);
      const iter = ms.subAlignmentSpanIterator(0n, 4);
      expect(Array.from(iter)).toEqual([1, 2, 3, 4]);
    });
  });

  describe("traverseAlignment", () => {
    it("should traverse alignment across multiple series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        alignment: 1n,
      });
      const b = new Series({
        data: new Float32Array([5, 6, 7]),
        alignment: 5n,
      });
      const multi = new MultiSeries([a, b]);
      expect(multi.traverseAlignment(1n, 2n)).toEqual(3n);
      expect(multi.traverseAlignment(2n, 4n)).toEqual(7n);
      expect(multi.traverseAlignment(5n, 2n)).toEqual(7n);
    });

    it("should handle empty multi-series", () => {
      const multi = new MultiSeries();
      expect(multi.traverseAlignment(1n, 2n)).toEqual(1n);
    });
  });

  describe("acquire and release", () => {
    it("should acquire and release all series in the multi-series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const b = new Series({
        data: new Float32Array([4, 5, 6]),
        dataType: DataType.FLOAT32,
      });
      const multi = new MultiSeries([a, b]);
      const controller = new MockGLBufferController();

      multi.acquire(controller);
      expect(a.refCount).toEqual(1);
      expect(b.refCount).toEqual(1);
      expect(controller.createBufferMock).toHaveBeenCalledTimes(2);

      multi.release();
      expect(a.refCount).toEqual(0);
      expect(b.refCount).toEqual(0);
      expect(controller.deleteBufferMock).toHaveBeenCalledTimes(2);
    });

    it("should handle empty multi-series", () => {
      const multi = new MultiSeries();
      const controller = new MockGLBufferController();
      multi.acquire(controller);
      multi.release();
    });
  });

  describe("distance", () => {
    it("should calculate distance between alignments across multiple series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        alignment: 1n,
      });
      const b = new Series({
        data: new Float32Array([5, 6, 7]),
        alignment: 5n,
      });
      const multi = new MultiSeries([a, b]);

      expect(multi.distance(1n, 3n)).toEqual(2n);
      expect(multi.distance(2n, 6n)).toEqual(3n);
      expect(multi.distance(5n, 7n)).toEqual(2n);
    });

    it("should handle empty multi-series", () => {
      const multi = new MultiSeries();
      expect(multi.distance(1n, 2n)).toEqual(0n);
    });

    it("should handle zero distance", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        alignment: 1n,
      });
      const multi = new MultiSeries([a]);
      expect(multi.distance(1n, 1n)).toEqual(0n);
    });
  });

  describe("byteLength", () => {
    it("should return the sum of byte lengths of all series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const b = new Series({
        data: new Float32Array([4, 5, 6]),
        dataType: DataType.FLOAT32,
      });
      const multi = new MultiSeries([a, b]);
      expect(multi.byteLength).toEqual(Size.bytes(24)); // 12 bytes per series (3 * 4 bytes)
    });

    it("should return 0 for empty multi-series", () => {
      const multi = new MultiSeries();
      expect(multi.byteLength).toEqual(Size.bytes(0));
    });
  });

  describe("updateGLBuffer", () => {
    it("should update GL buffers for all series", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const b = new Series({
        data: new Float32Array([4, 5, 6]),
        dataType: DataType.FLOAT32,
      });
      const multi = new MultiSeries([a, b]);
      const controller = new MockGLBufferController();

      multi.updateGLBuffer(controller);

      expect(controller.createBufferMock).toHaveBeenCalledTimes(2);
      expect(controller.bindBufferMock).toHaveBeenCalledTimes(2);
      expect(controller.bufferDataMock).toHaveBeenCalledTimes(2);
    });

    it("should handle empty multi-series", () => {
      const multi = new MultiSeries();
      const controller = new MockGLBufferController();

      multi.updateGLBuffer(controller);

      expect(controller.createBufferMock).not.toHaveBeenCalled();
      expect(controller.bindBufferMock).not.toHaveBeenCalled();
      expect(controller.bufferDataMock).not.toHaveBeenCalled();
    });

    it("should throw error for non-FLOAT32/UINT8 series", () => {
      const a = new Series({
        data: new BigInt64Array([BigInt(1), BigInt(2)]),
        dataType: DataType.INT64,
      });
      const multi = new MultiSeries([a]);
      const controller = new MockGLBufferController();

      expect(() => multi.updateGLBuffer(controller)).toThrow(
        "Only FLOAT32 and UINT8 arrays can be used in WebGL",
      );
    });

    it("should handle series with different buffer states", () => {
      const a = new Series({
        data: new Float32Array([1, 2, 3]),
        dataType: DataType.FLOAT32,
      });
      const b = Series.alloc({ capacity: 10, dataType: DataType.FLOAT32 });
      const multi = new MultiSeries([a, b]);
      const controller = new MockGLBufferController();

      multi.updateGLBuffer(controller);

      expect(controller.createBufferMock).toHaveBeenCalledTimes(2);
      expect(controller.bindBufferMock).toHaveBeenCalledTimes(1);
      expect(controller.bufferDataMock).toHaveBeenCalledTimes(1);
    });
  });

  describe("push", () => {
    it("should push a single Series to an empty MultiSeries", () => {
      const series = new Series({ data: [1, 2, 3], dataType: DataType.FLOAT32 });
      const multiSeries = new MultiSeries();
      multiSeries.push(series);
      expect(multiSeries.series.length).toBe(1);
      expect(multiSeries.series[0]).toBe(series);
    });

    it("should push a single Series to a non-empty MultiSeries", () => {
      const series1 = new Series({ data: [1, 2, 3], dataType: DataType.FLOAT32 });
      const series2 = new Series({ data: [4, 5, 6], dataType: DataType.FLOAT32 });
      const multiSeries = new MultiSeries([series1]);
      multiSeries.push(series2);
      expect(multiSeries.series.length).toBe(2);
      expect(multiSeries.series[0]).toBe(series1);
      expect(multiSeries.series[1]).toBe(series2);
    });

    it("should push a MultiSeries to another MultiSeries", () => {
      const series1 = new Series({ data: [1, 2, 3], dataType: DataType.FLOAT32 });
      const series2 = new Series({ data: [4, 5, 6], dataType: DataType.FLOAT32 });
      const multiSeries1 = new MultiSeries([series1]);
      const multiSeries2 = new MultiSeries([series2]);
      multiSeries1.push(multiSeries2);
      expect(multiSeries1.series.length).toBe(2);
      expect(multiSeries1.series[0]).toBe(series1);
      expect(multiSeries1.series[1]).toBe(series2);
    });

    it("should maintain data type consistency when pushing series", () => {
      const series1 = new Series({ data: [1, 2, 3], dataType: DataType.FLOAT32 });
      const series2 = new Series({ data: [4, 5, 6], dataType: DataType.FLOAT32 });
      const multiSeries = new MultiSeries([series1]);
      multiSeries.push(series2);
      expect(multiSeries.dataType).toEqual(DataType.FLOAT32);
    });

    it("should push an empty MultiSeries to a non-empty MultiSeries", () => {
      const series1 = new Series({ data: [1, 2, 3], dataType: DataType.FLOAT32 });
      const multiSeries1 = new MultiSeries([series1]);
      const multiSeries2 = new MultiSeries();
      multiSeries1.push(multiSeries2);
      expect(multiSeries1.series.length).toBe(1);
    });

    it("should push an empty MultiSeries to an empty MultiSeries", () => {
      const multiSeries1 = new MultiSeries();
      const multiSeries2 = new MultiSeries();
      multiSeries1.push(multiSeries2);
      expect(multiSeries1.series.length).toBe(0);
    });

    it("should update time range when pushing series", () => {
      const timeRange1 = new TimeRange(1, 2);
      const timeRange2 = new TimeRange(2, 3);
      const series1 = new Series({
        data: [1, 2, 3],
        dataType: DataType.FLOAT32,
        timeRange: timeRange1,
      });
      const series2 = new Series({
        data: [4, 5, 6],
        dataType: DataType.FLOAT32,
        timeRange: timeRange2,
      });
      const multiSeries = new MultiSeries([series1]);
      multiSeries.push(series2);
      expect(multiSeries.timeRange.start.valueOf()).toBe(1n);
      expect(multiSeries.timeRange.end.valueOf()).toBe(3n);
    });

    it("should throw an error when pushing a series with a different data type", () => {
      const series1 = new Series({ data: [1, 2, 3], dataType: DataType.FLOAT32 });
      const series2 = new Series({ data: [4, 5, 6], dataType: DataType.INT64 });
      const multiSeries = new MultiSeries([series1]);
      expect(() => multiSeries.push(series2)).toThrow(
        "cannot push a int64 series to a float32 multi-series",
      );
    });

    it("should throw an error when pushing a multi-series with a different data type", () => {
      const series1 = new Series({ data: [1, 2, 3], dataType: DataType.FLOAT32 });
      const series2 = new Series({ data: [4, 5, 6], dataType: DataType.INT64 });
      const multiSeries1 = new MultiSeries([series1]);
      const multiSeries2 = new MultiSeries([series2]);
      expect(() => multiSeries1.push(multiSeries2)).toThrow(
        "cannot push a int64 series to a float32 multi-series",
      );
    });
  });
});
