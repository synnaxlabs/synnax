// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { box } from "@/spatial";
import { type location } from "@/spatial/location";
import { position } from "@/spatial/position";

type Spec = [position.DialogProps, location.XY];

describe("position", () => {
  const SPEC_CASE_1: Spec = [
    {
      container: box.construct(0, 0, 100, 100),
      target: box.construct(45, 55, 10, 10),
      dialog: box.construct(0, 0, 20, 20),
    },
    { x: "left", y: "center" },
  ];

  const SPEC_CASE_2: Spec = [
    {
      container: box.construct(0, 0, 100, 100),
      target: box.construct(45, 55, 10, 10),
      dialog: box.construct(0, 0, 20, 20),
      initial: "top",
    },
    { x: "center", y: "top" },
  ];

  const SPEC_CASE_3: Spec = [
    {
      container: box.construct(0, 0, 100, 100),
      target: box.construct(45, 55, 10, 10),
      dialog: box.construct(0, 0, 20, 20),
      initial: "bottom",
    },
    { x: "center", y: "bottom" },
  ];

  const SPEC_CASE_4: Spec = [
    {
      container: box.construct(0, 0, 100, 100),
      target: box.construct(45, 55, 10, 10),
      dialog: box.construct(0, 0, 20, 20),
      initial: "left",
    },
    { x: "left", y: "center" },
  ];

  const SPEC_CASE_5: Spec = [
    {
      container: box.construct(0, 0, 100, 100),
      target: box.construct(45, 55, 10, 10),
      dialog: box.construct(0, 0, 20, 20),
      initial: "right",
    },
    { x: "right", y: "center" },
  ];

  // Target is in bottom right corner
  const SPEC_CASE_6: Spec = [
    {
      container: box.construct(0, 0, 100, 100),
      target: box.construct(90, 90, 10, 10),
      dialog: box.construct(0, 0, 20, 20),
    },
    { x: "left", y: "center" },
  ];

  const SPECS: Spec[] = [
    // SPEC_CASE_1,
    // SPEC_CASE_2,
    // SPEC_CASE_3,
    // SPEC_CASE_4,
    // SPEC_CASE_5,
    SPEC_CASE_6,
  ];

  describe("dialog", () => {
    SPECS.forEach(([props, expected]) => {
      it(`should position dialog correctly`, () => {
        expect(position.dialog(props)).toEqual(expected);
      });
    });
  });
});
