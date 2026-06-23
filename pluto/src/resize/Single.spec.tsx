// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Resize } from "@/resize";

// Testing the Resize is extremely difficult on a unit level
// using jsdom, so we're just going to test that it renders
// for now.
describe("Resize", () => {
  describe("Resize", () => {
    it("should render a resize div", async () => {
      const sizeBounds: bounds.Bounds = { lower: 20, upper: 500 };
      const c = render(
        <Resize.Single
          location="left"
          size={50}
          sizeBounds={sizeBounds}
          onResize={() => {}}
        >
          <p>Hello</p>
        </Resize.Single>,
      );
      expect(c.getByText("Hello")).toBeTruthy();
    });
  });
});
