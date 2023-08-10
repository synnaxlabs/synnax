// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test, it } from "vitest";

import { MockGLBufferController } from "@/mock/MockGLBufferController";
import { Series } from "@/telem/series";
import { DataType, Rate, Size, TimeRange, TimeStamp } from "@/telem/telem";

describe("Series", () => {
  describe("construction", () => {
    test("valid from native", () => {
      const a = new Series(new Float32Array([1, 2, 3]));
      expect(a.dataType.toString()).toBe(DataType.FLOAT32.toString());
      expect(a.length).toEqual(3);
      expect(a.byteLength).toEqual(Size.bytes(12));
      expect(a.byteCap).toEqual(Size.bytes(12));
      expect(a.cap).toEqual(3);
      const b = new Series(new BigInt64Array([BigInt(1)]));
      expect(b.dataType.toString()).toBe(DataType.INT64.toString());
      const c = new Series(new BigInt64Array([BigInt(1)]), DataType.TIMESTAMP);
      expect(c.dataType.toString()).toBe(DataType.TIMESTAMP.toString());
    });

    test("from buffer without data type provided", () => {
      expect(() => {
        // eslint-disable-next-line no-new
        new Series(new ArrayBuffer(4));
      }).toThrow();
    });

    test("from buffer with data type provided", () => {
      const a = new Series(new ArrayBuffer(4), DataType.FLOAT32);
      expect(a.dataType.toString()).toBe(DataType.FLOAT32.toString());
    });

    test("with time range", () => {
      const a = new Series(
        new Float32Array([1, 2, 3]),
        DataType.FLOAT32,
        new TimeRange(1, 2)
      );
      expect(a.timeRange.span.valueOf()).toBe(1);
    });

    describe("allocation", () => {
      it("should allocate a lazy array", () => {
        const arr = Series.alloc(10, DataType.FLOAT32);
        expect(arr.byteCap).toEqual(Size.bytes(40));
        expect(arr.cap).toEqual(10);
        expect(arr.length).toEqual(0);
        expect(arr.byteLength).toEqual(Size.bytes(0));
      });
      it("should throw an error when attempting to allocate an array of lenght 0", () => {
        expect(() => {
          Series.alloc(0, DataType.FLOAT32);
        }).toThrow();
      });
    });
  });

  test("at", () => {
    it("should return the value at the given index and add the sample offset", () => {
      const arr = new Series(
        new Float32Array([1, 2, 3]),
        DataType.FLOAT32,
        undefined,
        2
      );
      expect(arr.at(0)).toEqual(3);
      expect(arr.at(1)).toEqual(4);
      expect(arr.at(2)).toEqual(5);
    });
    it("should return undefined when the index is out of bounds", () => {
      const arr = new Series(new Float32Array([1, 2, 3]), DataType.FLOAT32);
      expect(arr.at(3)).toBeUndefined();
    });
  });

  describe("slice", () => {
    it("should slice a lazy array", () => {
      const a = new Series(new Float32Array([1, 2, 3]), DataType.FLOAT32);
      const b = a.slice(1, 2);
      expect(b.dataType.toString()).toBe(DataType.FLOAT32.toString());
      expect(b.data).toEqual(new Float32Array([2]));
      expect(b.length).toEqual(1);
      expect(b.byteLength).toEqual(Size.bytes(4));
      expect(b.byteCap).toEqual(Size.bytes(4));
      expect(b.cap).toEqual(1);
    });
  });

  describe("min and max", () => {
    it("should return a min and max of zero on an allocated array", () => {
      const arr = Series.alloc(10, DataType.FLOAT32);
      expect(arr.max).toEqual(-Infinity);
      expect(arr.min).toEqual(Infinity);
    });
    it("should correctly calculate the min and max of a lazy array", () => {
      const arr = new Series(new Float32Array([1, 2, 3]), DataType.FLOAT32);
      expect(arr.max).toEqual(3);
      expect(arr.min).toEqual(1);
    });
  });

  describe("conversion", () => {
    test("from float64 to float32", () => {
      const a = new Series(new Float64Array([1, 2, 3]), DataType.FLOAT64);
      const b = a.convert(DataType.FLOAT32);
      expect(b.dataType.toString()).toBe(DataType.FLOAT32.toString());
      expect(b.data).toEqual(new Float32Array([1, 2, 3]));
    });

    test("from int64 to int32", () => {
      const a = new Series(new BigInt64Array([BigInt(1), BigInt(2), BigInt(3)]));
      const b = a.convert(DataType.INT32);
      expect(b.dataType.toString()).toBe(DataType.INT32.toString());
      expect(b.data).toEqual(new Int32Array([1, 2, 3]));
    });

    test("from float32 to int64", () => {
      const a = new Series(new Float32Array([1, 2, 3]), DataType.FLOAT32);
      const b = a.convert(DataType.INT64);
      expect(b.dataType.toString()).toBe(DataType.INT64.toString());
      expect(b.data).toEqual(new BigInt64Array([BigInt(1), BigInt(2), BigInt(3)]));
    });
  });

  describe("writing", () => {
    it("should correctly write to an allocated lazy array", () => {
      const arr = Series.alloc(10, DataType.FLOAT32);
      expect(arr.byteCap).toEqual(Size.bytes(40));
      expect(arr.length).toEqual(0);
      const writeOne = new Series(new Float32Array([1]));
      expect(arr.write(writeOne)).toEqual(1);
      expect(arr.length).toEqual(1);
      const writeTwo = new Series(new Float32Array([2, 3]));
      expect(arr.write(writeTwo)).toEqual(2);
      expect(arr.length).toEqual(3);
    });
    it("should recompute cached max and min correctly", () => {
      const arr = Series.alloc(10, DataType.FLOAT32);
      arr.enrich();
      const writeTwo = new Series(new Float32Array([2, 3]));
      arr.write(writeTwo);
      expect(arr.max).toEqual(3);
      expect(arr.min).toEqual(2);
    });
    it("should correctly adjust the sample offset of a written array", () => {
      const arr = Series.alloc(2, DataType.FLOAT32, TimeRange.ZERO, -3);
      const writeOne = new Series(new Float32Array([-2]));
      expect(arr.write(writeOne)).toEqual(1);
      expect(arr.min).toEqual(-5);
      const writeTwo = new Series(
        new Float32Array([1]),
        DataType.FLOAT32,
        TimeRange.ZERO,
        -1
      );
      expect(arr.write(writeTwo)).toEqual(1);
      expect(arr.min).toEqual(-5);
      expect(arr.max).toEqual(-2);
    });
  });

  describe("generateTimeStamps", () => {
    it("should correctly generate timestamps", () => {
      const ts = Series.generateTimestamps(5, Rate.hz(1), TimeStamp.seconds(1));
      expect(ts.timeRange).toEqual(
        new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(6))
      );
      expect(ts.cap).toEqual(5);
      expect(ts.length).toEqual(5);
      expect(ts.dataType.toString()).toEqual(DataType.TIMESTAMP.toString());
      expect(ts.data).toEqual(
        new BigInt64Array([
          BigInt(TimeStamp.seconds(1).valueOf()),
          BigInt(TimeStamp.seconds(2).valueOf()),
          BigInt(TimeStamp.seconds(3).valueOf()),
          BigInt(TimeStamp.seconds(4).valueOf()),
          BigInt(TimeStamp.seconds(5).valueOf()),
        ])
      );
    });
  });

  describe("webgl buffering", () => {
    it("should correctly buffer a new lazy array", () => {
      const arr = new Series(new Float32Array([1, 2, 3]), DataType.FLOAT32);
      const controller = new MockGLBufferController();
      arr.updateGLBuffer(controller);
      expect(controller.createBufferMock).toHaveBeenCalledTimes(1);
      expect(controller.bindBufferMock).toHaveBeenCalledTimes(1);
      expect(controller.bufferDataMock).toHaveBeenCalledTimes(1);
      expect(controller.bufferSubDataMock).not.toHaveBeenCalled();
      const buf = controller.buffers[arr.glBuffer as number];
      expect(buf).toBeDefined();
      expect(buf.byteLength).toEqual(12);
      expect(buf).toEqual(new Float32Array([1, 2, 3]));
    });
    it("should correctly update a buffer when writing to an allocated array", () => {
      const arr = Series.alloc(10, DataType.FLOAT32);
      const controller = new MockGLBufferController();
      arr.updateGLBuffer(controller);
      expect(controller.createBufferMock).toHaveBeenCalledTimes(1);
      expect(controller.bindBufferMock).not.toHaveBeenCalled();
      expect(controller.bufferDataMock).not.toHaveBeenCalled();
      expect(controller.bufferSubDataMock).not.toHaveBeenCalled();
      let buf = controller.buffers[arr.glBuffer as number];
      expect(buf).toBeDefined();
      expect(buf.byteLength).toEqual(0);
      const writeOne = new Series(new Float32Array([1]));
      arr.write(writeOne);
      arr.updateGLBuffer(controller);
      expect(controller.bufferDataMock).toHaveBeenCalledTimes(1);
      expect(controller.bufferSubDataMock).toHaveBeenCalledTimes(1);
      buf = controller.buffers[arr.glBuffer as number];
      expect(buf.byteLength).toEqual(arr.byteCap.valueOf());
      expect(new Float32Array(buf)[0]).toEqual(1);
      const writeTwo = new Series(new Float32Array([2, 3]));
      arr.write(writeTwo);
      arr.updateGLBuffer(controller);
      expect(controller.bufferDataMock).not.toHaveBeenCalledTimes(2);
      expect(controller.bufferSubDataMock).toHaveBeenCalledTimes(2);
      buf = controller.buffers[arr.glBuffer as number];
      expect(buf.byteLength).toEqual(arr.byteCap.valueOf());
      expect(new Float32Array(buf)[0]).toEqual(1);
      expect(new Float32Array(buf)[1]).toEqual(2);
      expect(new Float32Array(buf)[2]).toEqual(3);
    });
  });

  describe("acquire", () => {
    it("should increase the reference count and buffer gl data", () => {
      const s = new Series(new Float32Array([1, 2, 3]));
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
});
