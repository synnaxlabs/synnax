// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeRange } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { framer } from "@/framer";

describe("framer.Frame", () => {
  describe("construction", () => {
    describe("valid", () => {
      test("from an array of channel names and an array of arrays", () => {
        const f = new framer.Frame(
          ["a", "b", "c"],
          [
            new Series({ data: new Float32Array([1, 2, 3]) }),
            new Series({ data: new Float32Array([1, 2, 3]) }),
            new Series({ data: new Float32Array([1, 2, 3]) }),
          ],
        );
        expect(f.length).toEqual(9);
        expect(f.colType).toEqual("name");
      });

      test("from an array of channel keys and an array of arrays", () => {
        const f = new framer.Frame(
          [12, 13, 14],
          [
            new Series({ data: new Float32Array([1, 2, 3]) }),
            new Series({ data: new Float32Array([1, 2, 3]) }),
            new Series({ data: new Float32Array([1, 2, 3]) }),
          ],
        );
        expect(f.length).toEqual(9);
        expect(f.colType).toEqual("key");
      });

      test("from a single name and an array of arrays", () => {
        const f = new framer.Frame("a", [
          new Series({ data: new Float32Array([1, 2, 3]) }),
        ]);
        expect(f.length).toEqual(3);
        expect(f.colType).toEqual("name");
      });

      test("from a single key and an array of arrays", () => {
        const f = new framer.Frame(12, [
          new Series({ data: new Float32Array([1, 2, 3]) }),
        ]);
        expect(f.length).toEqual(3);
        expect(f.colType).toEqual("key");
      });

      test("from a single key and a single array", () => {
        const f = new framer.Frame(
          12,
          new Series({ data: new Float32Array([1, 2, 3]) }),
        );
        expect(f.length).toEqual(3);
        expect(f.colType).toEqual("key");
      });

      test("from a single name and a single array", () => {
        const f = new framer.Frame(
          "a",
          new Series({ data: new Float32Array([1, 2, 3]) }),
        );
        expect(f.length).toEqual(3);
        expect(f.colType).toEqual("name");
      });

      describe("payload parsing", () => {
        it("should correctly parse a series payload", () => {
          const f = new framer.Frame({
            keys: [12],
            series: [{ dataType: new DataType("float32"), data: new ArrayBuffer(12) }],
          });
          expect(f.length.valueOf()).toEqual(3);
          expect(f.columns.length).toEqual(1);
          expect(f.series.length).toEqual(1);
        });

        it("should correctly parse a series with null data", () => {
          const f = new framer.Frame({
            keys: [12],
            series: [{ dataType: "float32", data: null }],
          });
          expect(f.length.valueOf()).toEqual(0);
          expect(f.columns.length).toEqual(1);
          expect(f.series.length).toEqual(1);
        });
      });

      test("from record", () => {
        const f = new framer.Frame({
          a: new Series({ data: new Float32Array([1, 2, 3]) }),
        });
        expect(f.length.valueOf()).toEqual(3);
        expect(f.columns.length).toEqual(1);
        expect(f.series.length).toEqual(1);
      });

      test("from map", () => {
        const f = new framer.Frame(
          new Map([[12, new Series({ data: new Float32Array([1, 2, 3]) })]]),
        );
        expect(f.length).toEqual(3);
        expect(f.columns.length).toEqual(1);
        expect(f.series.length).toEqual(1);
      });
    });

    describe("invalid", () => {
      test("mismatched lengths", () => {
        expect(
          () =>
            new framer.Frame(
              ["a", "b", "c"],
              [
                new Series({ data: new Float32Array([1, 2, 3]) }),
                new Series({ data: new Float32Array([1, 2, 3]) }),
              ],
            ),
        ).toThrow();
      });
    });
  });

  describe("vertical", () => {
    it("should return false if a key has more than one array", () => {
      const f = new framer.Frame(
        new Map([
          [12, [new Series({ data: new Float32Array([1, 2, 3]) })]],
          [
            13,
            [
              new Series({ data: new Float32Array([1, 2, 3]) }),
              new Series({ data: new Float32Array([1, 2, 3]) }),
            ],
          ],
        ]),
      );
      expect(f.isVertical).toEqual(false);
    });
  });

  describe("horizontal", () => {
    it("should return false if there is more than one key", () => {
      const f = new framer.Frame(
        new Map([
          [12, [new Series({ data: new Float32Array([1, 2, 3]) })]],
          [13, [new Series({ data: new Float32Array([1, 2, 3]) })]],
        ]),
      );
      expect(f.isHorizontal).toEqual(false);
    });
  });

  describe("weaklyAligned", () => {
    it("should return true if all keys have the same time range", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(500, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(500, 50000),
              }),
            ],
          ],
        ]),
      );
      expect(f.isWeaklyAligned).toEqual(true);
    });

    it("should return false if any key has a different time range", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(500, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      expect(f.isWeaklyAligned).toEqual(false);
    });
  });

  describe("timeRange", () => {
    describe("no key provided", () => {
      it("should return the maximum time range of the frame", () => {
        const f = new framer.Frame(
          new Map([
            [
              12,
              [
                new Series({
                  data: new Float32Array([1, 2, 3]),
                  timeRange: new TimeRange(40, 50000),
                }),
              ],
            ],
            [
              13,
              [
                new Series({
                  data: new Float32Array([1, 2, 3]),
                  timeRange: new TimeRange(500, 50001),
                }),
              ],
            ],
          ]),
        );
        expect(f.timeRange()).toEqual(new TimeRange(40, 50001));
      });
    });

    describe("key provided", () => {
      it("should return the time range of the key", () => {
        const f = new framer.Frame({
          a: new Series({
            data: new Float32Array([1, 2, 3]),
            timeRange: new TimeRange(40, 50000),
          }),
          b: new Series({
            data: new Float32Array([1, 2, 3]),
            timeRange: new TimeRange(500, 50001),
          }),
        });
        expect(f.timeRange("a")).toEqual(new TimeRange(40, 50000));
      });
    });

    describe("filter", () => {
      it("should return a frame filtered on a particular condition", () => {
        const f = new framer.Frame(
          new Map([
            [
              12,
              [
                new Series({
                  data: new Float32Array([1, 2, 3]),
                  timeRange: new TimeRange(40, 50000),
                }),
              ],
            ],
            [
              13,
              [
                new Series({
                  data: new Float32Array([1, 2, 3]),
                  timeRange: new TimeRange(500, 50001),
                }),
              ],
            ],
          ]),
        );
        expect(f.filter((k) => k === 12).columns).toEqual([12]);
      });
    });
  });

  describe("toPayload", () => {
    it("should return the frame as framer.FramePayload", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      const pld = f.toPayload();
      expect(pld.keys).toEqual([12, 13]);
      expect(pld.series?.[0].data.byteLength).toEqual(12);
    });
  });

  describe("latest", () => {
    it("should return the latest sample from each column in the frame", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      expect(f.latest()).toEqual({ 12: 3, 13: 3 });
    });
    it("should return the latest sample for each col in the frame - even with multiple series per col", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
              new Series({
                data: new Float32Array([4, 5, 6]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(500, 50001),
              }),
              new Series({
                data: new Float32Array([4, 5, 7]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      expect(f.latest()).toEqual({ 12: 6, 13: 7 });
    });

    it("should not add a key if no samples exist for the channel", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
          [14, []],
        ]),
      );
      expect(f.latest()).toEqual({ 12: 3, 13: 3 });
    });
  });

  describe("sample access", () => {
    it("should return the sample at the given index", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      expect(f.get(12).at(0)).toEqual(1);
    });
  });

  describe("at", () => {
    it("should return the sample at the given index", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      expect(f.at(0)).toEqual({ 12: 1, 13: 1 });
    });
    it("should throw an error if required is true and the index is out of bounds", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      expect(() => f.at(3, true)).toThrow();
    });
    it("should return undefined if required is false and the index is out of bounds", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([1, 2]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      expect(f.at(2)).toEqual({ 12: 3, 13: undefined });
    });
  });

  describe("digest", () => {
    it("should return digest information about the frame", () => {
      const s1 = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: new TimeRange(40, 50000),
      });
      const s2 = new Series({
        data: new Float32Array([4, 5, 6]),
        timeRange: new TimeRange(50001, 60000),
      });
      const s3 = new Series({
        data: new Float32Array([7, 8, 9]),
        timeRange: new TimeRange(500, 50001),
      });
      const f = new framer.Frame(
        new Map([
          [12, [s1, s2]],
          [13, [s3]],
        ]),
      );
      const digest = f.digest;
      expect(Object.keys(digest)).toEqual(["12", "13"]);
      expect(digest[12].length).toEqual(2);
      expect(digest[13].length).toEqual(1);
      expect(digest[12][0]).toEqual(s1.digest);
      expect(digest[12][1]).toEqual(s2.digest);
      expect(digest[13][0]).toEqual(s3.digest);
    });
  });

  describe("mapFilter", () => {
    it("should filter out items based on the keep boolean", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([4, 5, 6]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
          [
            14,
            [
              new Series({
                data: new Float32Array([7, 8, 9]),
                timeRange: new TimeRange(600, 60000),
              }),
            ],
          ],
        ]),
      );
      const filtered = f.mapFilter((k, arr) => [k, arr, k !== 13]);
      expect(filtered.columns).toEqual([12, 14]);
      expect(filtered.series.length).toEqual(2);
    });

    it("should map keys to new values", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([4, 5, 6]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      const mapped = f.mapFilter((k, arr) => {
        const newKey = typeof k === "number" ? k + 100 : k;
        return [newKey, arr, true];
      });
      expect(mapped.columns).toEqual([112, 113]);
      expect(mapped.series.length).toEqual(2);
    });

    it("should map series to new series", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
        ]),
      );
      const mapped = f.mapFilter((k, arr) => {
        const newData = new Float32Array(arr.length);
        for (let j = 0; j < arr.length; j++) newData[j] = (arr.at(j) as number) * 2;
        const newArr = new Series({
          data: newData,
          timeRange: arr.timeRange,
        });
        return [k, newArr, true];
      });
      expect(mapped.get(12).at(0)).toEqual(2);
      expect(mapped.get(12).at(1)).toEqual(4);
      expect(mapped.get(12).at(2)).toEqual(6);
    });

    it("should both map and filter simultaneously", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([4, 5, 6]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
          [
            14,
            [
              new Series({
                data: new Float32Array([7, 8, 9]),
                timeRange: new TimeRange(600, 60000),
              }),
            ],
          ],
        ]),
      );
      const result = f.mapFilter((k, arr) => {
        const keep = typeof k === "number" && k % 2 === 0;
        const newKey = typeof k === "number" ? k + 1 : k;
        return [newKey, arr, keep];
      });
      expect(result.columns).toEqual([13, 15]);
      expect(result.series.length).toEqual(2);
    });

    it("should handle empty frames", () => {
      const f = new framer.Frame();
      const result = f.mapFilter((k, arr) => [k, arr, true]);
      expect(result.columns).toEqual([]);
      expect(result.series.length).toEqual(0);
    });

    it("should handle filtering out all items", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([4, 5, 6]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      const result = f.mapFilter((k, arr) => [k, arr, false]);
      expect(result.columns).toEqual([]);
      expect(result.series.length).toEqual(0);
    });

    it("should keep all items when keep is always true", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([4, 5, 6]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      const result = f.mapFilter((k, arr) => [k, arr, true]);
      expect(result.columns).toEqual([12, 13]);
      expect(result.series.length).toEqual(2);
    });

    it("should use the index parameter correctly", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([4, 5, 6]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
          [
            14,
            [
              new Series({
                data: new Float32Array([7, 8, 9]),
                timeRange: new TimeRange(600, 60000),
              }),
            ],
          ],
        ]),
      );
      // Keep only items at even indices
      const result = f.mapFilter((k, arr, i) => [k, arr, i % 2 === 0]);
      expect(result.columns).toEqual([12, 14]);
      expect(result.series.length).toEqual(2);
    });

    it("should work with string keys", () => {
      const f = new framer.Frame({
        a: new Series({
          data: new Float32Array([1, 2, 3]),
          timeRange: new TimeRange(40, 50000),
        }),
        b: new Series({
          data: new Float32Array([4, 5, 6]),
          timeRange: new TimeRange(500, 50001),
        }),
        c: new Series({
          data: new Float32Array([7, 8, 9]),
          timeRange: new TimeRange(600, 60000),
        }),
      });
      const result = f.mapFilter((k, arr) => {
        const newKey = k === "a" ? "x" : k;
        const keep = k === "a" || k === "c";
        return [newKey, arr, keep];
      });
      expect(result.columns).toEqual(["x", "c"]);
      expect(result.series.length).toEqual(2);
    });

    it("should handle frames with multiple series per channel", () => {
      const f = new framer.Frame(
        new Map([
          [
            12,
            [
              new Series({
                data: new Float32Array([1, 2, 3]),
                timeRange: new TimeRange(40, 50000),
              }),
              new Series({
                data: new Float32Array([4, 5, 6]),
                timeRange: new TimeRange(50001, 60000),
              }),
            ],
          ],
          [
            13,
            [
              new Series({
                data: new Float32Array([7, 8, 9]),
                timeRange: new TimeRange(500, 50001),
              }),
            ],
          ],
        ]),
      );
      const result = f.mapFilter((k, arr) => [k, arr, k === 12]);
      expect(result.columns).toEqual([12, 12]);
      expect(result.series.length).toEqual(2);
    });
  });
});
