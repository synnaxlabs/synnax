// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { box } from "@/spatial";
import { type Alignment } from "@/spatial/base";
import { location } from "@/spatial/location";
import { position } from "@/spatial/position";

type Spec = [position.DialogProps, location.XY];

describe("position", () => {
  describe("dialog", () => {
    const SPEC_CASE_1: Spec = [
      {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(45, 55, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
      },
      location.LEFT_CENTER,
    ];

    const SPEC_CASE_2: Spec = [
      {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(45, 55, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        initial: "top",
      },
      location.TOP_CENTER,
    ];

    const SPEC_CASE_3: Spec = [
      {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(45, 55, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        initial: "bottom",
      },
      location.BOTTOM_CENTER,
    ];

    const SPEC_CASE_4: Spec = [
      {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(45, 55, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        initial: "left",
      },
      location.LEFT_CENTER,
    ];

    const SPEC_CASE_5: Spec = [
      {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(45, 55, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        initial: "right",
      },
      location.RIGHT_CENTER,
    ];

    // Target is in bottom right corner
    const SPEC_CASE_6: Spec = [
      {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(90, 90, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
      },
      location.LEFT_CENTER,
    ];

    // Target is in the top left corner
    const SPEC_CASE_7: Spec = [
      {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(0, 0, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
      },
      location.BOTTOM_CENTER,
    ];

    const SPEC_CASE_8: Spec = [
      {
        container: {
          one: { x: 0, y: 0 },
          two: { x: 1707, y: 1075 },
          root: { x: "left", y: "top" },
        },
        target: {
          one: { x: 79, y: 965 },
          two: { x: 1647, y: 992 },
          root: { x: "left", y: "top" },
        },
        dialog: {
          one: { x: 79.53125, y: 781 },
          two: { x: 1647.53125, y: 1141 },
          root: { x: "left", y: "top" },
        },
        initial: { x: "center" },
        alignments: ["center"],
      },
      location.TOP_CENTER,
    ];

    const SPEC_CASE_9: Spec = [
      {
        container: {
          one: { x: 0, y: 0 },
          two: { x: 1707, y: 697 },
          root: { x: "left", y: "top" },
        },
        target: {
          one: { x: 79, y: 587 },
          two: { x: 1647, y: 614 },
          root: { x: "left", y: "top" },
        },
        dialog: {
          one: { x: 79, y: 991 },
          two: { x: 1647, y: 1351 },
          root: { x: "left", y: "top" },
        },
        initial: { x: "center" },
        alignments: ["center"],
      },
      location.TOP_CENTER,
    ];

    // const SPEC_CASE_10 = [
    //   {
    //     container: {
    //       one: { x: 0, y: 0 },
    //       two: { x: 1707, y: 697 },
    //       root: { x: "left", y: "top" },
    //     },
    //     target: {
    //       one: { x: 78, y: 4 },
    //       two: { x: 278, y: 31 },
    //       root: { x: "left", y: "top" },
    //     },
    //     dialog: {
    //       one: { x: 78, y: 4 },
    //       two: { x: 78, y: 4 },
    //       root: { x: "left", y: "top" },
    //     },
    //     alignments: ["end"],
    //     disable: ["center"],
    //   },
    //   location.BOTTOM_LEFT,
    // ];

    const SPECS: Spec[] = [
      SPEC_CASE_1,
      SPEC_CASE_2,
      SPEC_CASE_3,
      SPEC_CASE_4,
      SPEC_CASE_5,
      SPEC_CASE_6,
      SPEC_CASE_7,
      SPEC_CASE_8,
      SPEC_CASE_9,
      // SPEC_CASE_10,
    ];

    SPECS.forEach(([props, expected]) => {
      it(`should position dialog correctly`, () => {
        expect(position.dialog(props).location).toEqual(expected);
      });
    });
  });
  describe("getRoot", () => {
    const SPECS: Array<[location.XY, Alignment, location.XY]> = [
      [location.TOP_LEFT, "start", location.BOTTOM_RIGHT],
      [location.TOP_LEFT, "center", location.BOTTOM_CENTER],
      [location.TOP_LEFT, "end", location.BOTTOM_LEFT],
      [location.TOP_RIGHT, "start", location.BOTTOM_LEFT],
      [location.TOP_RIGHT, "center", location.BOTTOM_CENTER],
      [location.TOP_RIGHT, "end", location.BOTTOM_RIGHT],
      [location.TOP_CENTER, "start", location.BOTTOM_LEFT],
      [location.TOP_CENTER, "center", location.BOTTOM_CENTER],
      [location.TOP_CENTER, "end", location.BOTTOM_RIGHT],
      [location.LEFT_CENTER, "start", location.BOTTOM_RIGHT],
      [location.LEFT_CENTER, "center", location.RIGHT_CENTER],
      [location.LEFT_CENTER, "end", location.TOP_RIGHT],
      [location.RIGHT_CENTER, "start", location.BOTTOM_LEFT],
      [location.RIGHT_CENTER, "center", location.LEFT_CENTER],
      [location.RIGHT_CENTER, "end", location.TOP_LEFT],
      [location.BOTTOM_LEFT, "start", location.TOP_RIGHT],
      [location.BOTTOM_LEFT, "center", location.TOP_CENTER],
      [location.BOTTOM_LEFT, "end", location.TOP_LEFT],
      [location.BOTTOM_RIGHT, "start", location.TOP_LEFT],
      [location.BOTTOM_RIGHT, "center", location.TOP_CENTER],
      [location.BOTTOM_RIGHT, "end", location.TOP_RIGHT],
      [location.BOTTOM_CENTER, "start", location.TOP_LEFT],
      [location.BOTTOM_CENTER, "center", location.TOP_CENTER],
      [location.BOTTOM_CENTER, "end", location.TOP_RIGHT],
    ];
    SPECS.forEach(([option, order, expected]) => {
      it(`should position get the correct positioning root for ${location.xyToString(option)} and ${order}`, () => {
        expect(position.getRoot(option, order)).toEqual(expected);
      });
    });
  });
});
