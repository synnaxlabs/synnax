// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Orientation } from "@/schematic/node/common/orientation";

const ZERO_VALUE: Orientation.Value = { inner: "top", outer: "top" };

const buttons = (container: HTMLElement): HTMLButtonElement[] =>
  Array.from(container.querySelectorAll("button"));

describe("Orientation.Select", () => {
  describe("default layout (4 outer + 4 inner)", () => {
    it("should render exactly eight buttons", () => {
      const { container } = render(
        <Orientation.Select value={ZERO_VALUE} onChange={() => {}} />,
      );
      expect(buttons(container)).toHaveLength(8);
    });

    // Button order is deterministic from JSX:
    // [0] outer top, [1] outer left, [2] inner top (dir-y),
    // [3] inner left, [4] inner right, [5] inner bottom (dir-y),
    // [6] outer right, [7] outer bottom
    it.each<[string, number, Partial<Orientation.Value>]>([
      ["outer top", 0, { outer: "top" }],
      ["outer left", 1, { outer: "left" }],
      ["inner top", 2, { inner: "top" }],
      ["inner left", 3, { inner: "left" }],
      ["inner right", 4, { inner: "right" }],
      ["inner bottom", 5, { inner: "bottom" }],
      ["outer right", 6, { outer: "right" }],
      ["outer bottom", 7, { outer: "bottom" }],
    ])(
      "should call onChange with the merged value when %s is clicked",
      (_name, index, expectedDelta) => {
        const onChange = vi.fn();
        const { container } = render(
          <Orientation.Select
            value={{ inner: "right", outer: "left" }}
            onChange={onChange}
          />,
        );
        fireEvent.click(buttons(container)[index]);
        expect(onChange).toHaveBeenCalledTimes(1);
        expect(onChange).toHaveBeenCalledWith({
          inner: "right",
          outer: "left",
          ...expectedDelta,
        });
      },
    );

    it("should mark inner top with the y-direction class", () => {
      const { container } = render(
        <Orientation.Select value={ZERO_VALUE} onChange={() => {}} />,
      );
      expect(buttons(container)[2].className).toContain("pluto--direction-y");
    });

    it("should mark inner bottom with the y-direction class", () => {
      const { container } = render(
        <Orientation.Select value={ZERO_VALUE} onChange={() => {}} />,
      );
      expect(buttons(container)[5].className).toContain("pluto--direction-y");
    });
  });

  describe("selected highlighting", () => {
    it("should add the selected modifier to the matching outer button", () => {
      const { container } = render(
        <Orientation.Select
          value={{ inner: "top", outer: "right" }}
          onChange={() => {}}
        />,
      );
      // outer right is index 6 in the default layout
      expect(buttons(container)[6].className).toContain("pluto--selected");
      // outer top should not be selected
      expect(buttons(container)[0].className).not.toContain("pluto--selected");
    });

    it("should add the selected modifier to the matching inner button", () => {
      const { container } = render(
        <Orientation.Select
          value={{ inner: "left", outer: "top" }}
          onChange={() => {}}
        />,
      );
      // inner left is index 3 in the default layout
      expect(buttons(container)[3].className).toContain("pluto--selected");
    });

    it("should highlight at most one outer and one inner button", () => {
      const { container } = render(
        <Orientation.Select
          value={{ inner: "right", outer: "bottom" }}
          onChange={() => {}}
        />,
      );
      const selected = buttons(container).filter((b) =>
        b.className.includes("pluto--selected"),
      );
      expect(selected).toHaveLength(2);
    });
  });

  describe("hideOuter", () => {
    it("should render only the inner cluster", () => {
      const { container } = render(
        <Orientation.Select value={ZERO_VALUE} onChange={() => {}} hideOuter />,
      );
      // Internal renders 4 inner buttons (no outer buttons, no center).
      expect(buttons(container)).toHaveLength(4);
    });

    it("should still call onChange with inner-targeted updates", () => {
      const onChange = vi.fn();
      const { container } = render(
        <Orientation.Select
          value={{ inner: "top", outer: "bottom" }}
          onChange={onChange}
          hideOuter
        />,
      );
      // [0] inner top, [1] inner left, [2] inner right, [3] inner bottom
      fireEvent.click(buttons(container)[1]);
      expect(onChange).toHaveBeenCalledWith({
        inner: "left",
        outer: "bottom",
      });
    });
  });

  describe("hideInner", () => {
    it("should mark the inner buttons as disabled and leave outer buttons enabled", () => {
      const { container } = render(
        <Orientation.Select value={ZERO_VALUE} onChange={() => {}} hideInner />,
      );
      const all = buttons(container);
      // Pluto's Button uses a CSS class instead of the HTML disabled attribute,
      // since the rendered <button> still needs to be focus-traversable but
      // not click-actionable.
      [2, 3, 4, 5].forEach((i) =>
        expect(all[i].className).toContain("pluto--disabled"),
      );
      [0, 1, 6, 7].forEach((i) =>
        expect(all[i].className).not.toContain("pluto--disabled"),
      );
    });

    it("should hide inner buttons visually via the hidden-inner modifier class", () => {
      const { container } = render(
        <Orientation.Select value={ZERO_VALUE} onChange={() => {}} hideInner />,
      );
      const innerLeft = buttons(container)[3];
      expect(innerLeft.className).toContain("pluto--hidden-inner");
    });

    it("should not fire onChange for clicks on disabled inner buttons", () => {
      const onChange = vi.fn();
      const { container } = render(
        <Orientation.Select value={ZERO_VALUE} onChange={onChange} hideInner />,
      );
      fireEvent.click(buttons(container)[3]);
      expect(onChange).not.toHaveBeenCalled();
    });
  });

  describe("showOuterCenter", () => {
    it("should replace the four inner buttons with a single center button", () => {
      const { container } = render(
        <Orientation.Select value={ZERO_VALUE} onChange={() => {}} showOuterCenter />,
      );
      // 4 outer + 1 center = 5
      expect(buttons(container)).toHaveLength(5);
    });

    it("should set outer to center when the center button is clicked", () => {
      const onChange = vi.fn();
      const { container } = render(
        <Orientation.Select
          value={{ inner: "top", outer: "left" }}
          onChange={onChange}
          showOuterCenter
        />,
      );
      // [0] outer top, [1] outer left, [2] center, [3] outer right, [4] outer bottom
      fireEvent.click(buttons(container)[2]);
      expect(onChange).toHaveBeenCalledWith({ inner: "top", outer: "center" });
    });

    it("should mark the center button selected when outer is center", () => {
      const { container } = render(
        <Orientation.Select
          value={{ inner: "top", outer: "center" }}
          onChange={() => {}}
          showOuterCenter
        />,
      );
      expect(buttons(container)[2].className).toContain("pluto--selected");
    });

    it("should add the show-outer-center modifier to the inner wrapper", () => {
      const { container } = render(
        <Orientation.Select value={ZERO_VALUE} onChange={() => {}} showOuterCenter />,
      );
      expect(container.querySelector(".pluto-value")?.className).toContain(
        "pluto--show-outer-center",
      );
    });
  });

  describe("class hooks", () => {
    it("should always tag the root with pluto-select-orientation", () => {
      const { container } = render(
        <Orientation.Select value={ZERO_VALUE} onChange={() => {}} />,
      );
      expect(container.querySelector(".pluto-select-orientation")).not.toBeNull();
    });

    it("should still tag the root with pluto-select-orientation when hideOuter is set", () => {
      const { container } = render(
        <Orientation.Select value={ZERO_VALUE} onChange={() => {}} hideOuter />,
      );
      expect(container.querySelector(".pluto-select-orientation")).not.toBeNull();
    });
  });
});
