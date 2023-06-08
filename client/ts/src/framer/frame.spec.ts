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
    describe("valid", () => {
      test("from an array of channel names and an array of arrays", () => {
        const f = new Frame(
          ["a", "b", "c"],
          [
            new LazyArray(new Float32Array([1, 2, 3])),
            new LazyArray(new Float32Array([1, 2, 3])),
            new LazyArray(new Float32Array([1, 2, 3])),
          ]
        );
        expect(f.length).toEqual(9);
        expect(f.labeledBy).toEqual("name");
      });

      test("from an array of channel keys and an array of arrays", () => {
        const f = new Frame(
          [12, 13, 14],
          [
            new LazyArray(new Float32Array([1, 2, 3])),
            new LazyArray(new Float32Array([1, 2, 3])),
            new LazyArray(new Float32Array([1, 2, 3])),
          ]
        );
        expect(f.length).toEqual(9);
        expect(f.labeledBy).toEqual("key");
      });

      test("from a single name and an array of arrays", () => {
        const f = new Frame("a", [new LazyArray(new Float32Array([1, 2, 3]))]);
        expect(f.length).toEqual(3);
        expect(f.labeledBy).toEqual("name");
      });

      test("from a single key and an array of arrays", () => {
        const f = new Frame(12, [new LazyArray(new Float32Array([1, 2, 3]))]);
        expect(f.length).toEqual(3);
        expect(f.labeledBy).toEqual("key");
      });

      test("from a single key and a single array", () => {
        const f = new Frame(12, new LazyArray(new Float32Array([1, 2, 3])));
        expect(f.length).toEqual(3);
        expect(f.labeledBy).toEqual("key");
      });

      test("from a single name and a single array", () => {
        const f = new Frame("a", new LazyArray(new Float32Array([1, 2, 3])));
        expect(f.length).toEqual(3);
        expect(f.labeledBy).toEqual("name");
      });

      test("from payload", () => {
        const f = new Frame({
          keys: [12],
          arrays: [
            {
              dataType: new DataType("float32"),
              data: new SharedArrayBuffer(12),
            },
          ],
        });
        expect(f.length.valueOf()).toEqual(3);
        expect(f.labels.length).toEqual(1);
        expect(f.arrays.length).toEqual(1);
      });

      test("from record", () => {
        const f = new Frame({
          a: new LazyArray(new Float32Array([1, 2, 3])),
        });
        expect(f.length.valueOf()).toEqual(3);
        expect(f.labels.length).toEqual(1);
        expect(f.arrays.length).toEqual(1);
      });

      test("from map", () => {
        const f = new Frame(
          new Map([[12, new LazyArray(new Float32Array([1, 2, 3]))]])
        );
        expect(f.length).toEqual(3);
        expect(f.labels.length).toEqual(1);
        expect(f.arrays.length).toEqual(1);
      });
    });

    describe("invalid", () => {
      test("mismatched lengths", () => {
        expect(
          () =>
            new Frame(
              ["a", "b", "c"],
              [
                new LazyArray(new Float32Array([1, 2, 3])),
                new LazyArray(new Float32Array([1, 2, 3])),
              ]
            )
        ).toThrow();
      });
    });
  });

  describe("vertical", () => {
    it("should return false if a key has more than one array", () => {
      const f = new Frame(
        new Map([
          [12, [new LazyArray(new Float32Array([1, 2, 3]))]],
          [
            13,
            [
              new LazyArray(new Float32Array([1, 2, 3])),
              new LazyArray(new Float32Array([1, 2, 3])),
            ],
          ],
        ])
      );
      expect(f.isVertical).toEqual(false);
    });
  });

  describe("horizontal", () => {
    it("should return false if there is more than one key", () => {
      const f = new Frame(
        new Map([
          [12, [new LazyArray(new Float32Array([1, 2, 3]))]],
          [13, [new LazyArray(new Float32Array([1, 2, 3]))]],
        ])
      );
      expect(f.isHorizontal).toEqual(false);
    });
  });

  describe("weaklyAligned", () => {
    it("should return true if all keys have the same timerange", () => {
      const f = new Frame(
        new Map([
          [
            12,
            [
              new LazyArray(
                new Float32Array([1, 2, 3]),
                undefined,
                new TimeRange(500, 50000)
              ),
            ],
          ],
          [
            13,
            [
              new LazyArray(
                new Float32Array([1, 2, 3]),
                undefined,
                new TimeRange(500, 50000)
              ),
            ],
          ],
        ])
      );
      expect(f.isWeaklyAligned).toEqual(true);
    });

    it("should return false if any key has a different timerange", () => {
      const f = new Frame(
        new Map([
          [
            12,
            [
              new LazyArray(
                new Float32Array([1, 2, 3]),
                undefined,
                new TimeRange(500, 50000)
              ),
            ],
          ],
          [
            13,
            [
              new LazyArray(
                new Float32Array([1, 2, 3]),
                undefined,
                new TimeRange(500, 50001)
              ),
            ],
          ],
        ])
      );
      expect(f.isWeaklyAligned).toEqual(false);
    });
  });

  describe("timeRange", () => {
    describe("no key provided", () => {
      it("should return the maxium time range of the frame", () => {
        const f = new Frame(
          new Map([
            [
              12,
              [
                new LazyArray(
                  new Float32Array([1, 2, 3]),
                  undefined,
                  new TimeRange(40, 50000)
                ),
              ],
            ],
            [
              13,
              [
                new LazyArray(
                  new Float32Array([1, 2, 3]),
                  undefined,
                  new TimeRange(500, 50001)
                ),
              ],
            ],
          ])
        );
        expect(f.timeRange()).toEqual(new TimeRange(40, 50001));
      });
    });

    describe("key provided", () => {
      it("should return the time range of the key", () => {
        const f = new Frame({
          a: new LazyArray(
            new Float32Array([1, 2, 3]),
            undefined,
            new TimeRange(40, 50000)
          ),
          b: new LazyArray(
            new Float32Array([1, 2, 3]),
            undefined,
            new TimeRange(500, 50001)
          ),
        });
        expect(f.timeRange("a")).toEqual(new TimeRange(40, 50000));
      });
    });

    describe("filter", () => {
      it("should return a frame filtered on a particular condition", () => {
        const f = new Frame(
          new Map([
            [
              12,
              [
                new LazyArray(
                  new Float32Array([1, 2, 3]),
                  undefined,
                  new TimeRange(40, 50000)
                ),
              ],
            ],
            [
              13,
              [
                new LazyArray(
                  new Float32Array([1, 2, 3]),
                  undefined,
                  new TimeRange(500, 50001)
                ),
              ],
            ],
          ])
        );
        expect(f.filter((k) => k === 12).labels).toEqual([12]);
      });
    });
  });

  describe("toPayload", () => {
    it("should return the frame as FramePayload", () => {
      const f = new Frame(
        new Map([
          [
            12,
            [
              new LazyArray(
                new Float32Array([1, 2, 3]),
                undefined,
                new TimeRange(40, 50000)
              ),
            ],
          ],
          [
            13,
            [
              new LazyArray(
                new Float32Array([1, 2, 3]),
                undefined,
                new TimeRange(500, 50001)
              ),
            ],
          ],
        ])
      );
      const pld = f.toPayload();
      expect(pld.keys).toEqual([12, 13]);
      expect(pld.arrays?.[0].data.byteLength).toEqual(12);
    });
  });
});
