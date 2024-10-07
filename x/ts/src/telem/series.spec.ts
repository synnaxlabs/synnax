// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";
import { z } from "zod";

import { MockGLBufferController } from "@/mock/MockGLBufferController";
import { isCrudeSeries, MultiSeries, Series } from "@/telem/series";
import { DataType, Rate, Size, TimeRange, TimeSpan, TimeStamp } from "@/telem/telem";

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

    it("should convert encoded keys to snake_case", () => {
      const a = new Series({ data: [{ aB: 1, bC: "apple" }], dataType: DataType.JSON });
      const strContent = new TextDecoder().decode(a.data);
      expect(strContent).toBe('{"a_b":1,"b_c":"apple"}\n');
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
    it("should throw an error if that data type is not numeric", () => {
      const series = new Series({ data: ["a", "b", "c"] });
      expect(() => series.max == null).toThrow();
      expect(() => series.min == null).toThrow();
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

  describe("writing", () => {
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
      series.enrich();
      const writeTwo = new Series({ data: new Float32Array([2, 3]) });
      series.write(writeTwo);
      expect(series.max).toEqual(3);
      expect(series.min).toEqual(2);
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

  describe("generateTimeStamps", () => {
    it("should correctly generate timestamps", () => {
      const ts = Series.generateTimestamps(5, Rate.hz(1), TimeStamp.seconds(1));
      expect(ts.timeRange).toEqual(
        new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(6)),
      );
      expect(ts.capacity).toEqual(5);
      expect(ts.length).toEqual(5);
      expect(ts.dataType.toString()).toEqual(DataType.TIMESTAMP.toString());
      expect(ts.data).toEqual(
        new BigInt64Array([
          BigInt(TimeStamp.seconds(1).valueOf()),
          BigInt(TimeStamp.seconds(2).valueOf()),
          BigInt(TimeStamp.seconds(3).valueOf()),
          BigInt(TimeStamp.seconds(4).valueOf()),
          BigInt(TimeStamp.seconds(5).valueOf()),
        ]),
      );
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
      expect(buf).toEqual(new Float32Array([1, 2, 3]));
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
      const s = Series.fromStrings(["apple", "banana", "carrot"]);
      expect(s.dataType.toString()).toEqual(DataType.STRING.toString());
      const outStrings = s.toStrings();
      expect(outStrings).toEqual(["apple", "banana", "carrot"]);
    });
    it("should throw an error if the series is not of type string", () => {
      const s = new Series({ data: new Float32Array([1, 2, 3]) });
      expect(() => {
        s.toStrings();
      }).toThrow();
    });
    it("should not throw an error if the series is of type UUID", () => {
      const s = new Series({
        data: new Uint8Array([1, 2, 3]),
        dataType: DataType.UUID,
      });
      expect(() => {
        s.toStrings();
      }).not.toThrow();
    });
    it("should return an array of length 0 if the series is empty", () => {
      const s = new Series({ data: new Float32Array([]), dataType: DataType.STRING });
      const outStrings = s.toStrings();
      expect(outStrings).toEqual([]);
    });
  });

  describe("JSON series", () => {
    it("should correctly encode and decode a JSON series", () => {
      const schema = z.object({
        a: z.number(),
        b: z.string(),
      });
      const s = Series.fromJSON<z.output<typeof schema>>([
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

  describe("digest", () => {
    it("should return a digest of information about the series", () => {
      const digest = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: new TimeRange(1, 3),
      }).digest;
      expect(digest.alignment).toEqual({ lower: 0n, upper: 3n });
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
});
