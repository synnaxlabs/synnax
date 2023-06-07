import { DataType, LazyArray } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { DynamicCache } from "./dynamic";

describe("DynamicReadCache", () => {
  it("Should correctly allocate a buffer", () => {
    const cache = new DynamicCache(100, DataType.FLOAT32);
    const arr = new LazyArray(new Float32Array([1, 2, 3]), DataType.FLOAT32);
    expect(cache.write([arr])).toHaveLength(0);
    expect(cache.length).toEqual(arr.length);
  });
  it("should correctly allocate a single new buffer when the current one is full", () => {
    const cache = new DynamicCache(2, DataType.FLOAT32);
    const arr = new LazyArray(new Float32Array([1, 2, 3]), DataType.FLOAT32);
    expect(cache.write([arr])).toHaveLength(1);
    expect(cache.length).toEqual(1);
  });
  it("should correctly allocate multiple new buffers when the current one is full", () => {
    const cache = new DynamicCache(1, DataType.FLOAT32);
    const arr = new LazyArray(new Float32Array([1, 2, 3]), DataType.FLOAT32);
    expect(cache.write([arr])).toHaveLength(2);
    expect(cache.length).toEqual(1);
  });
  it("it should correctly set multiple writes", () => {
    const cache = new DynamicCache(100, DataType.FLOAT32);
    const arr = new LazyArray(new Float32Array([1, 2, 3]), DataType.FLOAT32);
    expect(cache.write([arr])).toHaveLength(0);
    expect(cache.write([arr])).toHaveLength(0);
    expect(cache.write([arr])).toHaveLength(0);
    expect(cache.length).toEqual(arr.length * 3);
    expect(cache.data.data.slice(0, 3)).toEqual(new Float32Array([1, 2, 3]));
    expect(cache.data.data.slice(3, 6)).toEqual(new Float32Array([1, 2, 3]));
    expect(cache.data.data.slice(6, 9)).toEqual(new Float32Array([1, 2, 3]));
  });
});
