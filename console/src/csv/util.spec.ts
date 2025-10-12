// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, Frame, Series } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { convertFrameGroups, type CSVGroup } from "@/csv/util";

describe("csv.util.convertFrameGroups", () => {
  describe.only("valid", () => {
    interface ValidSpec {
      description: string;
      groups: CSVGroup[];
      expected: string;
    }
    const SPECS: ValidSpec[] = [
      {
        description: "should correctly convert a frame with multiple channels",
        groups: [
          {
            index: 12,
            frame: new Frame({
              12: new Series({ dataType: DataType.TIMESTAMP, data: [1, 2, 3] }),
              13: new Series({ dataType: DataType.FLOAT32, data: [10, 20, 30] }),
            }),
          },
        ],
        expected: `1,10
      2,20
      3,30
      `,
      },
      {
        description: "should correctly convert a single frame with a single channel",
        groups: [
          {
            index: 12,
            frame: new Frame({
              12: new Series({ dataType: DataType.TIMESTAMP, data: [1, 2, 3] }),
            }),
          },
        ],
        expected: `1
      2
      3
      `,
      },
      {
        description:
          "should correctly convert multiple frames with multiple channels that are not aligned",
        groups: [
          {
            index: 12,
            frame: new Frame({
              12: new Series({ dataType: DataType.TIMESTAMP, data: [1, 2, 3] }),
            }),
          },
          {
            index: 13,
            frame: new Frame({
              13: new Series({ dataType: DataType.TIMESTAMP, data: [10, 20, 30] }),
            }),
          },
        ],
        expected: `1,
2,
3,
,10
,20
,30
`,
      },
      {
        description:
          "should correctly convert a frame with multiple channels that have some alignment",
        groups: [
          {
            index: 12,
            frame: new Frame({
              12: new Series({ dataType: DataType.TIMESTAMP, data: [1, 2, 20, 40] }),
              13: new Series({ dataType: DataType.FLOAT32, data: [10, 20, 30, 40] }),
            }),
          },
          {
            index: 14,
            frame: new Frame({
              14: new Series({ dataType: DataType.TIMESTAMP, data: [10, 20, 30, 40] }),
              15: new Series({ dataType: DataType.FLOAT32, data: [10, 20, 30, 40] }),
            }),
          },
        ],
        expected: `1,
2,
3,
,10
,20
,30
`,
      },
    ];
    SPECS.forEach(({ description, groups, expected }) => {
      it(description, () => {
        const csv = convertFrameGroups(groups);
        expect(csv).toEqual(expected);
      });
    });
  });
  // describe.skip("invalid", () => {
  //   interface InvalidSpec {
  //     description: string;
  //     group: Map<channel.KeyOrName, Frame>;
  //     errorMessage: string;
  //   }
  //   const SPECS: InvalidSpec[] = [
  //     {
  //       description: "should throw an error if the groups have repeated keys",
  //       group: [
  //         {
  //           index: 12,
  //           frame: new Frame({ 12: new Series({ dataType: DataType.TIMESTAMP }) }),
  //         },
  //         {
  //           index: 13,
  //           13,
  //           frame: new Frame({
  //             12: new Series({ dataType: DataType.TIMESTAMP }),
  //             13: new Series({ dataType: DataType.TIMESTAMP }),
  //           }),
  //           ],
  //         },
  //       ],
  //       errorMessage: "Index 12 is repeated between frames",
  //     },
  //     {
  //       description: "should throw an error if the specified index is not found",
  //       group: new Map([["b", new Frame({ a: new Series([1, 2, 3]) })]]),
  //       errorMessage: "Frame for index b has no respective series",
  //     },
  //     {
  //       description: "should throw an error if the index series is not a timestamp",
  //       group: [
  //         {
  //           index: 12,
  //           frame: new Frame({ 12: new Series([1, 2, 3]) }),
  //         },
  //       ],
  //       errorMessage: "Index series for index 12 is not a timestamp",
  //     },
  //     {
  //       description: "should throw an error if a frame is not vertical",
  //       group: [
  //         {
  //           index: "channel1",
  //           frame: new Frame({ channel1: new Series([1, 2, 3]) }),
  //         },
  //       ],
  //         {
  //           index: "channel1",
  //           frame: new Frame({ channel1: new Series([1, 2, 3]) }),
  //         },
  //       ],
  //           frame: new Frame(
  //             {
  //               channel1: new Series([1, 2, 3]),
  //               channel1: new Series([4, 5, 6]),
  //             },
  //           ),
  //         },
  //       ],
  //             ["channel1", "channel1"],
  //             [
  //               new Series({ data: [1, 2, 3], dataType: DataType.TIMESTAMP }),
  //               new Series({ data: [4, 5, 6], dataType: DataType.TIMESTAMP }),
  //             ],
  //           ),
  //         },
  //       ],
  //             ["channel1", "channel1"],
  //             [
  //               new Series({ data: [1, 2, 3], dataType: DataType.TIMESTAMP }),
  //               new Series({ data: [4, 5, 6], dataType: DataType.TIMESTAMP }),
  //             ],
  //           ),
  //         },
  //       ],
  //             {
  //               channel1: new Series([1, 2, 3]),
  //               channel1: new Series([4, 5, 6]),
  //             },
  //           ),
  //         },
  //       ],
  //             ["channel1", "channel1"],
  //             [
  //               new Series({ data: [1, 2, 3], dataType: DataType.TIMESTAMP }),
  //               new Series({ data: [4, 5, 6], dataType: DataType.TIMESTAMP }),
  //             ],
  //           ),
  //         ],
  //       ],
  //       errorMessage: "Frame with index channel1 has repeated series with the same key",
  //     },
  //     {
  //       description:
  //         "should throw an error if a series is not the same length as the index series",
  //       group: [
  //         {
  //           12,
  //           new Frame({
  //             12: new Series({ dataType: DataType.TIMESTAMP, data: [1, 2, 3] }),
  //             13: new Series({ data: [1, 2, 3, 4] }),
  //           }),
  //         ],
  //       ],
  //       errorMessage:
  //         "Series for channel 13 is not the same length (4) as the series for index 12 (3)",
  //     },
  //   ];
  //   SPECS.forEach(({ description, group, errorMessage }) => {
  //     it(description, () => {
  //       expect(() => convertFrameGroups(group)).toThrow(errorMessage);
  //     });
  //   });
  // });
});
