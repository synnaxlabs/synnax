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
import { type ControlPosition, ReactFlowProvider } from "@xyflow/react";
import { type FC, type ReactElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Haul } from "@/haul";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type Primitive } from "@/schematic/node/common/primitive";
import { spec as circleSpec } from "@/schematic/node/general/circle/external";
import { spec as polygonSpec } from "@/schematic/node/general/polygon/external";
import { Theming } from "@/theming";

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

interface SpyResizeControlProps {
  position: ControlPosition;
  keepAspectRatio?: boolean;
  onResize?: (event: unknown, params: { width: number; height: number }) => void;
}

const { resizeControls } = vi.hoisted(() => ({
  resizeControls: new Map<string, SpyResizeControlProps>(),
}));

// NodeResizeControl renders for real; the spy only records keepAspectRatio and
// onResize so tests can invoke the resize callback the DOM does not expose.
vi.mock("@xyflow/react", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const Real = actual.NodeResizeControl as FC<SpyResizeControlProps>;
  return {
    ...actual,
    NodeResizeControl: (props: SpyResizeControlProps): ReactElement => {
      resizeControls.set(props.position, props);
      return <Real {...props} />;
    },
  };
});

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

interface ResizeTestConfig extends Label.LabeledConfig {
  radius: number;
}

const Dummy: FC<Primitive.SVGBasedProps> = () => <div data-testid="dummy-symbol" />;

const ResizeWrap = ({ children }: { children: ReactNode }): ReactElement => (
  <ReactFlowProvider>
    <Theming.Provider>
      <Haul.Provider>
        <div data-id={NODE_KEY}>{children}</div>
      </Haul.Provider>
    </Theming.Provider>
  </ReactFlowProvider>
);

describe("Label.createLabeled resize wiring", () => {
  beforeEach(() => resizeControls.clear());

  it("should not render resize controls without an onResize override", () => {
    const Node = Label.createLabeled<ResizeTestConfig>(Dummy);
    render(
      <ResizeWrap>
        <Node
          nodeKey={NODE_KEY}
          selected
          onConfigChange={vi.fn()}
          config={{ radius: 10 }}
        />
      </ResizeWrap>,
    );
    expect(resizeControls.size).toBe(0);
  });

  it("should map a resize to a config patch through the onResize override", () => {
    const onConfigChange = vi.fn();
    const Node = Label.createLabeled<ResizeTestConfig>(Dummy, {
      onResize: ({ width }) => ({ radius: width / 2 }),
    });
    render(
      <ResizeWrap>
        <Node
          nodeKey={NODE_KEY}
          selected
          onConfigChange={onConfigChange}
          config={{ radius: 10 }}
        />
      </ResizeWrap>,
    );
    resizeControls.get("right")?.onResize?.(null, { width: 40, height: 40 });
    expect(onConfigChange).toHaveBeenCalledWith({ radius: 20 });
  });

  it("should forward a grid override such as keepAspectRatio to the controls", () => {
    const Node = Label.createLabeled<ResizeTestConfig>(Dummy, {
      grid: { keepAspectRatio: true },
      onResize: ({ width }) => ({ radius: width }),
    });
    render(
      <ResizeWrap>
        <Node
          nodeKey={NODE_KEY}
          selected
          onConfigChange={vi.fn()}
          config={{ radius: 10 }}
        />
      </ResizeWrap>,
    );
    ["top", "right", "top-left", "bottom-right"].forEach((position) =>
      expect(resizeControls.get(position)?.keepAspectRatio).toBe(true),
    );
  });

  it("should resize a real circle by setting radius to half the width", () => {
    const onConfigChange = vi.fn();
    render(
      <ResizeWrap>
        <circleSpec.Node
          nodeKey={NODE_KEY}
          selected
          onConfigChange={onConfigChange}
          config={{ variant: "circle", radius: 20 }}
        />
      </ResizeWrap>,
    );
    resizeControls.get("right")?.onResize?.(null, { width: 50, height: 50 });
    expect(onConfigChange).toHaveBeenCalledWith({ radius: 25 });
  });

  it("should resize a real polygon by setting sideLength to half the width", () => {
    const onConfigChange = vi.fn();
    render(
      <ResizeWrap>
        <polygonSpec.Node
          nodeKey={NODE_KEY}
          selected
          onConfigChange={onConfigChange}
          config={{ variant: "polygon", numSides: 6, sideLength: 20 }}
        />
      </ResizeWrap>,
    );
    resizeControls.get("right")?.onResize?.(null, { width: 30, height: 30 });
    expect(onConfigChange).toHaveBeenCalledWith({ sideLength: 15 });
  });
});
