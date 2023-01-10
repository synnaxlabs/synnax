// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it, test } from "vitest";

import { distribute } from "./ResizeMultiple";

import { Resize } from ".";

const context = describe;

// Testing the Resize is extremely difficult on a unit level
// using jsdom, so we're just going to test that it renders
// for now.
describe("Resize", () => {
  describe("Resize", () => {
    it("should render a resize div", async () => {
      const c = render(
        <Resize location="left" initialSize={50} minSize={20} maxSize={500}>
          <p>Hello</p>
        </Resize>
      );
      expect(c.getByText("Hello")).toBeTruthy();
    });
  });
  describe("calculatePercentages", () => {
    context("sizes === count, sum of sizes equal to parent size", () => {
      it("should correctly distribute the sizes", () => {
        const sizes = [200, 200];
        const count = 2;
        const percentages = distribute(sizes, 400, count);
        expect(percentages).toHaveLength(2);
        expect(percentages[0]).toBeCloseTo(0.5);
        expect(percentages[1]).toBeCloseTo(0.5);
      });
    });
  });
});
