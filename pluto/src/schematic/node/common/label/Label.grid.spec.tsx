// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { Haul } from "@/haul";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";

const NODE_KEY = "n1";

const Wrap = ({ children }: { children: ReactNode }): ReactElement => (
  <Haul.Provider>
    <div data-id={NODE_KEY}>
      <Grid.Grid editable nodeKey={NODE_KEY}>
        {children}
      </Grid.Grid>
    </div>
  </Haul.Provider>
);

const slot = (container: HTMLElement, loc: location.Location): HTMLElement | null =>
  container.querySelector(`.pluto-grid__item.pluto--location-${loc}`);

const labelEl = (container: HTMLElement): HTMLElement | null =>
  container.querySelector(".pluto-symbol__label");

describe("Label.Label as GridItem", () => {
  describe("conditional rendering", () => {
    it("should render nothing when config is undefined", () => {
      const { container } = render(
        <Wrap>
          <Label.Label />
        </Wrap>,
      );
      expect(labelEl(container)).toBeNull();
    });

    it("should render nothing when label is undefined", () => {
      const { container } = render(
        <Wrap>
          <Label.Label config={{ orientation: "top" }} />
        </Wrap>,
      );
      expect(labelEl(container)).toBeNull();
    });

    it("should render nothing when label is the empty string", () => {
      const { container } = render(
        <Wrap>
          <Label.Label config={{ label: "" }} />
        </Wrap>,
      );
      expect(labelEl(container)).toBeNull();
    });

    it("should render the label text when a non-empty label is provided", () => {
      const { container } = render(
        <Wrap>
          <Label.Label config={{ label: "Hello", orientation: "top" }} />
        </Wrap>,
      );
      const el = labelEl(container);
      expect(el).not.toBeNull();
      expect(el?.textContent).toBe("Hello");
    });
  });

  describe("placement", () => {
    it("should default to the top slot when orientation is unset", () => {
      const { container } = render(
        <Wrap>
          <Label.Label config={{ label: "X" }} />
        </Wrap>,
      );
      expect(slot(container, "top")?.contains(labelEl(container))).toBe(true);
    });

    it.each<location.Outer>(["top", "right", "bottom", "left"])(
      "should sit in the %s slot when orientation is %s",
      (orientation) => {
        const { container } = render(
          <Wrap>
            <Label.Label config={{ label: "X", orientation }} />
          </Wrap>,
        );
        expect(slot(container, orientation)?.contains(labelEl(container))).toBe(true);
      },
    );
  });

  describe("style derivation", () => {
    it("should apply align as textAlign and forward maxInlineSize", () => {
      const { container } = render(
        <Wrap>
          <Label.Label config={{ label: "X", align: "start", maxInlineSize: 200 }} />
        </Wrap>,
      );
      const el = labelEl(container) as HTMLElement;
      expect(el.style.textAlign).toBe("start");
      expect(el.style.maxInlineSize).toBe("200px");
    });

    it("should apply the direction modifier class when direction is set", () => {
      const { container } = render(
        <Wrap>
          <Label.Label config={{ label: "X", direction: "y" }} />
        </Wrap>,
      );
      expect(labelEl(container)?.className).toContain("pluto--direction-y");
    });
  });

  describe("location change wiring", () => {
    it("should propagate a location change as a label config update", () => {
      const onChange = vi.fn();
      const { container } = render(
        <Wrap>
          <Label.Label
            config={{ label: "X", orientation: "top" }}
            onChange={onChange}
          />
        </Wrap>,
      );
      const labelDiv = labelEl(container) as HTMLElement;
      fireEvent.dragStart(labelDiv);
      fireEvent.dragOver(slot(container, "right") as HTMLElement, {
        screenX: 1,
        screenY: 1,
      });
      expect(onChange).toHaveBeenCalledTimes(1);
      expect(onChange).toHaveBeenCalledWith({
        label: { label: "X", orientation: "right" },
      });
    });
  });
});
