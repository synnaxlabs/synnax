// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, LazyArray, TimeRange } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { Frame } from "..";

describe("Frame", () => {
  describe("construction", () => {
    describe("from keys and arrays", () => {
      test("valid", () => {
        const f = new Frame([new LazyArray(new Float32Array([1, 2, 3]))], ["a"]);
        expect(f.size.valueOf()).toEqual(12);
        expect(f.keys.length).toEqual(1);
        expect(f.arrays.length).toEqual(1);
      });
      test("invalid", () => {
        expect(
          () => new Frame([new LazyArray(new Float32Array([1, 2, 3]))], [])
        ).toThrow();
        expect(() => new Frame([], ["a"])).toThrow();
      });
    });
    test("from payload", () => {
      const f = new Frame({
        keys: ["a"],
        arrays: [
          {
            dataType: new DataType("float32"),
            data: new SharedArrayBuffer(12),
          },
        ],
      });
      expect(f.size.valueOf()).toEqual(12);
      expect(f.keys.length).toEqual(1);
      expect(f.arrays.length).toEqual(1);
    });
    test("from record", () => {
      const f = new Frame({ a: [new LazyArray(new Float32Array([1, 2, 3]))] });
      expect(f.size.valueOf()).toEqual(12);
      expect(f.keys.length).toEqual(1);
      expect(f.arrays.length).toEqual(1);
    });
  });

  describe("vertical", () => {
    it("should return false if a key has more than one array", () => {
      const f = new Frame({
        a: [new LazyArray(new Float32Array([1, 2, 3]))],
        b: [
          new LazyArray(new Float32Array([1, 2, 3])),
          new LazyArray(new Float32Array([1, 2, 3])),
        ],
      });
      expect(f.isVertical).toEqual(false);
    });
  });

  describe("horizontal", () => {
    it("should return false if there is more than one key", () => {
      const f = new Frame({
        a: [new LazyArray(new Float32Array([1, 2, 3]))],
        b: [new LazyArray(new Float32Array([1, 2, 3]))],
      });
      expect(f.isHorizontal).toEqual(false);
    });
  });

  describe("weaklyAligned", () => {
    it("should return true if all keys have the same timerange", () => {
      const f = new Frame({
        a: [
          new LazyArray(
            new Float32Array([1, 2, 3]),
            undefined,
            new TimeRange(500, 50000)
          ),
        ],
        b: [
          new LazyArray(
            new Float32Array([1, 2, 3]),
            undefined,
            new TimeRange(500, 50000)
          ),
        ],
      });
      expect(f.isWeaklyAligned).toEqual(true);
    });

    it("should return false if any key has a different timerange", () => {
      const f = new Frame({
        a: [
          new LazyArray(
            new Float32Array([1, 2, 3]),
            undefined,
            new TimeRange(500, 50000)
          ),
        ],
        b: [
          new LazyArray(
            new Float32Array([1, 2, 3]),
            undefined,
            new TimeRange(500, 50001)
          ),
        ],
      });
      expect(f.isWeaklyAligned).toEqual(false);
    });
  });

  describe("timeRange", () => {
    describe("no key provided", () => {
      it("should return the maxium time range of the frame", () => {
        const f = new Frame({
          a: [
            new LazyArray(
              new Float32Array([1, 2, 3]),
              undefined,
              new TimeRange(40, 50000)
            ),
          ],
          b: [
            new LazyArray(
              new Float32Array([1, 2, 3]),
              undefined,
              new TimeRange(500, 50001)
            ),
          ],
        });
        expect(f.timeRange()).toEqual(new TimeRange(40, 50001));
      });
    });

    describe("key provided", () => {
      it("should return the time range of the key", () => {
        const f = new Frame({
          a: [
            new LazyArray(
              new Float32Array([1, 2, 3]),
              undefined,
              new TimeRange(40, 50000)
            ),
          ],
          b: [
            new LazyArray(
              new Float32Array([1, 2, 3]),
              undefined,
              new TimeRange(500, 50001)
            ),
          ],
        });
        expect(f.timeRange("a")).toEqual(new TimeRange(40, 50000));
      });
    });

    describe("filter", () => {
      it("should return a frame filtered on a particular condition", () => {
        const f = new Frame({
          a: [
            new LazyArray(
              new Float32Array([1, 2, 3]),
              undefined,
              new TimeRange(40, 50000)
            ),
          ],
          b: [
            new LazyArray(
              new Float32Array([1, 2, 3]),
              undefined,
              new TimeRange(500, 50001)
            ),
          ],
        });
        expect(f.filter((k) => k === "a").keys).toEqual(["a"]);
      });
    });
  });

  describe("toPayload", () => {
    it("should return the frame as FramePayload", () => {
      const f = new Frame({
        a: [
          new LazyArray(
            new Float32Array([1, 2, 3]),
            undefined,
            new TimeRange(40, 50000)
          ),
        ],
        b: [
          new LazyArray(
            new Float32Array([1, 2, 3]),
            undefined,
            new TimeRange(500, 50001)
          ),
        ],
      });
      const pld = f.toPayload();
      expect(pld.keys).toEqual(["a", "b"]);
      expect(pld.arrays?.[0].data.byteLength).toEqual(12);
    });
  });
});
