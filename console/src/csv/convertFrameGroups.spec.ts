// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Frame, Series } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { convertFrameGroups, type FrameGroup } from "@/csv/convertFrameGroups";

describe("csv", () => {
  describe("convertFrameGroups", () => {
    describe("valid", () => {
      it("should correctly convert a frame with multiple channels", () => {
        const s1 = new Series({ dataType: DataType.TIMESTAMP, data: [1, 2, 3] });
        const s2 = new Series({ dataType: DataType.FLOAT32, data: [10, 20, 30] });
        const groups: FrameGroup[] = [
          { index: 12, frame: new Frame({ 12: s1, 13: s2 }) },
        ];
        expect(convertFrameGroups(groups)).toEqual(`1,10
2,20
3,30
`);
      });
      it("should convert a frame with a single channel", () => {
        const series = new Series({ dataType: DataType.TIMESTAMP, data: [1, 2, 3] });
        const frame = new Frame(12, series);
        const groups: FrameGroup[] = [{ index: 12, frame }];
        expect(convertFrameGroups(groups)).toEqual(`1
2
3
`);
      });
      it("should correctly convert multiple frames with multiple channels that are not aligned", () => {
        const s1 = new Series({ dataType: DataType.TIMESTAMP, data: [10, 20, 30] });
        const s2 = new Series({ dataType: DataType.TIMESTAMP, data: [1, 2, 3] });
        const groups: FrameGroup[] = [
          { index: 13, frame: new Frame({ 13: s1 }) },
          { index: 12, frame: new Frame({ 12: s2 }) },
        ];
        expect(convertFrameGroups(groups)).toEqual(`,1
,2
,3
10,
20,
30,
`);
      });
      it("should work with multiple frames with multiple channels that are not aligned", () => {
        const g1s1 = new Series({ dataType: DataType.TIMESTAMP, data: [1, 6, 8] });
        const g1s2 = new Series({ dataType: DataType.FLOAT32, data: [10, 11, 12] });
        const g2s1 = new Series({ dataType: DataType.TIMESTAMP, data: [2, 4, 9] });
        const g2s2 = new Series({ dataType: DataType.FLOAT32, data: [20, 21, 22] });
        const g3s1 = new Series({ dataType: DataType.TIMESTAMP, data: [3, 5, 7] });
        const g3s2 = new Series({ dataType: DataType.FLOAT32, data: [30, 31, 32] });
        const groups: FrameGroup[] = [
          { index: 1, frame: new Frame({ 1: g1s1, 2: g1s2 }) },
          { index: 3, frame: new Frame({ 3: g2s1, 4: g2s2 }) },
          { index: 5, frame: new Frame({ 5: g3s1, 6: g3s2 }) },
        ];
        expect(convertFrameGroups(groups)).toEqual(`1,10,,,,
,,2,20,,
,,,,3,30
,,4,21,,
,,,,5,31
6,11,,,,
,,,,7,32
8,12,,,,
,,9,22,,
`);
      });
      it("should correctly work with empty series", () => {
        const emptySeries = new Series({ dataType: DataType.TIMESTAMP });
        const groups: FrameGroup[] = [
          { index: 1, frame: new Frame({ 1: emptySeries }) },
        ];
        expect(convertFrameGroups(groups)).toEqual("");
      });
      it("should work when a frame has several series for the same channel", () => {
        const s1 = new Series({ dataType: DataType.TIMESTAMP, data: [1, 2, 3] });
        const s2 = new Series({ dataType: DataType.FLOAT32, data: [10, 20, 30] });
        const frame = new Frame({ 1: s1, 2: s2 });
        frame.push(1, new Series({ dataType: DataType.TIMESTAMP, data: [4, 5, 6] }));
        frame.push(2, new Series({ dataType: DataType.FLOAT32, data: [7, 8, 9] }));
        const groups: FrameGroup[] = [{ index: 1, frame }];
        expect(convertFrameGroups(groups)).toEqual(`1,10
2,20
3,30
4,7
5,8
6,9
`);
      });
      it("should correctly work with empty series and filled series", () => {
        const emptySeries = new Series({ dataType: DataType.TIMESTAMP });
        const filledSeries = new Series({
          dataType: DataType.TIMESTAMP,
          data: [1, 2, 3],
        });
        const groups: FrameGroup[] = [
          { index: 1, frame: new Frame({ 1: emptySeries }) },
          {
            index: 2,
            frame: new Frame({ 2: filledSeries }),
          },
        ];
        expect(convertFrameGroups(groups)).toEqual(`,1
,2
,3
`);
      });
      it("should switch line endings if specified", () => {
        const s = new Series({ dataType: DataType.TIMESTAMP, data: [1, 2, 3] });
        const frame = new Frame({ 1: s });
        const groups: FrameGroup[] = [{ index: 1, frame }];
        expect(convertFrameGroups(groups, "\r\n")).toEqual("1\r\n2\r\n3\r\n");
        expect(convertFrameGroups(groups, "\n")).toEqual("1\n2\n3\n");
      });
      it("should correctly convert frames that have some alignment", () => {
        const g1s1 = new Series({ dataType: DataType.TIMESTAMP, data: [1, 3, 5] });
        const g1s2 = new Series({ dataType: DataType.FLOAT32, data: [100, 200, 300] });
        const g2s1 = new Series({ dataType: DataType.TIMESTAMP, data: [2, 3, 4] });
        const g2s2 = new Series({ dataType: DataType.FLOAT32, data: [10, 20, 30] });
        const groups: FrameGroup[] = [
          { index: 1, frame: new Frame({ 1: g1s1, 2: g1s2 }) },
          { index: 3, frame: new Frame({ 3: g2s1, 4: g2s2 }) },
        ];
        expect(convertFrameGroups(groups)).toEqual(`1,100,,
,,2,10
3,200,3,20
,,4,30
5,300,,
`);
      });
    });
    describe("invalid", () => {
      it("should throw an error if the groups have repeated keys", () => {
        const s = new Series({ dataType: DataType.TIMESTAMP });
        const groups: FrameGroup[] = [
          { index: 12, frame: new Frame({ 12: s }) },
          { index: 13, frame: new Frame({ 12: s, 13: s }) },
        ];
        expect(() => convertFrameGroups(groups)).toThrow(
          "Channel 12 is repeated between multiple frames",
        );
      });
      it("should throw an error if the specified index is not found", () => {
        const s = new Series({ dataType: DataType.TIMESTAMP });
        const groups: FrameGroup[] = [{ index: 13, frame: new Frame({ 12: s }) }];
        expect(() => convertFrameGroups(groups)).toThrow(
          "Index channel 13 is not of type timestamp",
        );
      });
      it("should throw an error if the index series is not a timestamp", () => {
        const s = new Series({ dataType: DataType.FLOAT32 });
        const groups: FrameGroup[] = [{ index: 12, frame: new Frame(12, s) }];
        expect(() => convertFrameGroups(groups)).toThrow(
          "Index channel 12 is not of type timestamp",
        );
      });
      it("should throw an error if a series is not the same length as the index series", () => {
        const s1 = new Series({ dataType: DataType.TIMESTAMP, data: [1, 2, 3] });
        const s2 = new Series({ dataType: DataType.FLOAT32, data: [1, 2, 3, 4] });
        const groups: FrameGroup[] = [
          { index: 12, frame: new Frame({ 12: s1, 13: s2 }) },
        ];
        expect(() => convertFrameGroups(groups)).toThrow(
          "Multi-series for channel 13 is not the same length (4) as the multi-series for index channel 12 (3)",
        );
      });
    });
  });
});
