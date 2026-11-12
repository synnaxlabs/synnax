// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Bounds } from "@/bounds";
import { mockBoundingClientRect } from "@/testutil/dom";

const RANGES: bounds.Bounds[] = [
  { lower: -10, upper: 10 },
  { lower: -5, upper: 5 },
  { lower: 0, upper: 5 },
];

describe("Bounds", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  describe("format", () => {
    it("should show bounds symmetric about zero with a plus-minus sign", () => {
      expect(Bounds.format({ lower: -10, upper: 10 }, "V")).toBe("±10 V");
    });

    it("should show other bounds as a span", () => {
      expect(Bounds.format({ lower: 0, upper: 0.2 })).toBe("0 to 0.2");
    });
  });

  describe("Select", () => {
    it("should show the bounds that equal the value", () => {
      const c = render(
        <Bounds.Select
          data={RANGES}
          value={{ lower: -5, upper: 5 }}
          onChange={vi.fn()}
          units="V"
        />,
      );
      expect(c.getByText("±5 V")).toBeTruthy();
    });

    it("should show the placeholder when the value equals no bounds", () => {
      const c = render(
        <Bounds.Select
          data={RANGES}
          value={{ lower: -3, upper: 3 }}
          onChange={vi.fn()}
          resourceName="input range"
        />,
      );
      expect(c.getByText("Input range")).toBeTruthy();
      expect(c.queryByText("±10")).toBeNull();
    });

    it("should call onChange with the bounds that the user picks", () => {
      const onChange = vi.fn();
      const c = render(
        <Bounds.Select
          data={RANGES}
          value={{ lower: -3, upper: 3 }}
          onChange={onChange}
          resourceName="input range"
        />,
      );
      fireEvent.click(c.getByText("Input range"));
      fireEvent.click(c.getByText("0 to 5"));
      expect(onChange).toHaveBeenCalledWith({ lower: 0, upper: 5 });
    });

    it("should list bounds that appear twice once", () => {
      const c = render(
        <Bounds.Select
          data={[...RANGES, { lower: 0, upper: 5 }]}
          value={{ lower: -3, upper: 3 }}
          onChange={vi.fn()}
          resourceName="input range"
        />,
      );
      fireEvent.click(c.getByText("Input range"));
      expect(c.getAllByText("0 to 5")).toHaveLength(1);
    });
  });
});
