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
import {
  NodeResizeControl,
  ReactFlowProvider,
  type ResizeControlProps,
  ResizeControlVariant,
  type ResizeDragEvent,
} from "@xyflow/react";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useState,
} from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Haul } from "@/haul";
import { Grid } from "@/schematic/node/common/grid";
import { Context as DiagramContext, ZERO_CONTEXT_VALUE } from "@/vis/diagram/Context";

interface RecordedControl {
  keepAspectRatio?: boolean;
  triggerResize: (width: number, height: number) => void;
}

const resizeControls = new Map<string, RecordedControl>();

const RESIZE_EVENT = {} as ResizeDragEvent;

// Renders the real NodeResizeControl so positions, variants, and gating are asserted
// on the DOM, and records a trigger so tests can drive a resize jsdom cannot perform.
const SpyResizeControl: FC<ResizeControlProps> = (props) => {
  resizeControls.set(props.position ?? "", {
    keepAspectRatio: props.keepAspectRatio,
    triggerResize: (width, height) =>
      props.onResize?.(RESIZE_EVENT, { x: 0, y: 0, width, height, direction: [] }),
  });
  return <NodeResizeControl {...props} />;
};

const diagramCtx = { ...ZERO_CONTEXT_VALUE, resizeControl: SpyResizeControl };

const NODE_KEY = "node-1";

interface GridHostProps extends Omit<Grid.GridProps, "children"> {
  children: ReactNode;
}

// Wraps Grid in a Haul.Provider so drag-and-drop has a context, and inside
// `data-id={nodeKey}` so reflowPane's selectNode lookup can resolve.
const GridHost = ({ children, ...rest }: GridHostProps): ReactElement => (
  <Haul.Provider>
    <div data-id={rest.nodeKey}>
      <Grid.Grid {...rest}>{children}</Grid.Grid>
    </div>
  </Haul.Provider>
);

const slot = (container: HTMLElement, loc: location.Location): HTMLElement | null =>
  container.querySelector(`.pluto-grid__item.pluto--location-${loc}`);

const dragHandle = (container: HTMLElement): HTMLElement | null =>
  container.querySelector(".pluto-drag-handle");

const rotateButton = (container: HTMLElement): HTMLElement | null =>
  container.querySelector(".pluto-grid__rotate");

describe("Grid item tagging", () => {
  it("should let createItem produce components that splitChildren resolves as items", () => {
    const Tagged = Grid.createItem<{ itemKey: string; loc: location.Location }>(
      ({ itemKey, loc }) => (
        <Grid.Item itemKey={itemKey} location={loc}>
          <div data-testid={`tagged-${itemKey}`}>{itemKey}</div>
        </Grid.Item>
      ),
    );
    const { container } = render(
      <GridHost editable={false} nodeKey={NODE_KEY}>
        <Tagged itemKey="a" loc="top" />
        <span data-testid="body-text">body content</span>
      </GridHost>,
    );
    const top = slot(container, "top");
    expect(top?.querySelector('[data-testid="tagged-a"]')).not.toBeNull();
    expect(
      dragHandle(container)?.querySelector('[data-testid="body-text"]'),
    ).not.toBeNull();
  });

  it("should treat untagged components as body content even when shaped like an item", () => {
    const Untagged = ({ itemKey }: { itemKey: string }) => (
      <Grid.Item itemKey={itemKey} location="top">
        <div data-testid={`untagged-${itemKey}`}>{itemKey}</div>
      </Grid.Item>
    );
    const { container } = render(
      <GridHost editable={false} nodeKey={NODE_KEY}>
        <Untagged itemKey="ghost" />
      </GridHost>,
    );
    expect(slot(container, "top")).toBeNull();
    expect(
      dragHandle(container)?.querySelector('[data-testid="untagged-ghost"]'),
    ).not.toBeNull();
  });

  it("should treat tagged components that do not return a Grid.Item as body content", () => {
    const TaggedButWrong = Grid.createItem<{ label: string }>(({ label }) => (
      <div data-testid="not-an-item">{label}</div>
    ));
    const { container } = render(
      <GridHost editable={false} nodeKey={NODE_KEY}>
        <TaggedButWrong label="hello" />
      </GridHost>,
    );
    // resolveItem returns null when the rendered element is not Grid.Item, and
    // splitChildren puts null-resolving children into `body`. So the original
    // TaggedButWrong component lands inside the drag-handle wrapper.
    const handle = dragHandle(container);
    expect(handle).not.toBeNull();
    expect(handle?.querySelector('[data-testid="not-an-item"]')).not.toBeNull();
    // It is not promoted into a slot.
    expect(slot(container, "top")).toBeNull();
  });
});

describe("Grid slot layout", () => {
  it("should render every edge slot in editable mode so they can act as drop targets", () => {
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY}>
        <div>only body</div>
      </GridHost>,
    );
    // EditableSlot does not bail when empty — each edge must exist as a drop
    // target so an item dragged from elsewhere can land on it.
    expect(slot(container, "top")).not.toBeNull();
    expect(slot(container, "right")).not.toBeNull();
    expect(slot(container, "bottom")).not.toBeNull();
    expect(slot(container, "left")).not.toBeNull();
  });

  it("should route each item into the slot that matches its location", () => {
    const { container } = render(
      <GridHost editable={false} nodeKey={NODE_KEY}>
        <Grid.Item itemKey="t" location="top">
          <div data-testid="t">T</div>
        </Grid.Item>
        <Grid.Item itemKey="r" location="right">
          <div data-testid="r">R</div>
        </Grid.Item>
        <Grid.Item itemKey="b" location="bottom">
          <div data-testid="b">B</div>
        </Grid.Item>
        <Grid.Item itemKey="l" location="left">
          <div data-testid="l">L</div>
        </Grid.Item>
      </GridHost>,
    );
    expect(slot(container, "top")?.querySelector('[data-testid="t"]')).not.toBeNull();
    expect(slot(container, "right")?.querySelector('[data-testid="r"]')).not.toBeNull();
    expect(
      slot(container, "bottom")?.querySelector('[data-testid="b"]'),
    ).not.toBeNull();
    expect(slot(container, "left")?.querySelector('[data-testid="l"]')).not.toBeNull();
  });

  it("should not render the center slot by default", () => {
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY}>
        <Grid.Item itemKey="c" location="center">
          <div data-testid="c">C</div>
        </Grid.Item>
      </GridHost>,
    );
    expect(slot(container, "center")).toBeNull();
    // and the item destined for center is not rendered anywhere
    expect(container.querySelector('[data-testid="c"]')).toBeNull();
  });

  it("should render the center slot when allowCenter is true", () => {
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY} allowCenter>
        <Grid.Item itemKey="c" location="center">
          <div data-testid="c">C</div>
        </Grid.Item>
      </GridHost>,
    );
    expect(
      slot(container, "center")?.querySelector('[data-testid="c"]'),
    ).not.toBeNull();
  });

  it("should render body content inside the drag-handle wrapper", () => {
    const { container } = render(
      <GridHost editable={false} nodeKey={NODE_KEY}>
        <Grid.Item itemKey="t" location="top">
          <div data-testid="t">T</div>
        </Grid.Item>
        <div data-testid="symbol">{"<symbol>"}</div>
      </GridHost>,
    );
    expect(
      dragHandle(container)?.querySelector('[data-testid="symbol"]'),
    ).not.toBeNull();
    // And the item is NOT inside the drag-handle.
    expect(dragHandle(container)?.querySelector('[data-testid="t"]')).toBeNull();
  });

  it("should not render an empty static slot when no items target that edge", () => {
    const { container } = render(
      <GridHost editable={false} nodeKey={NODE_KEY}>
        <Grid.Item itemKey="t" location="top">
          <div data-testid="t">T</div>
        </Grid.Item>
      </GridHost>,
    );
    expect(slot(container, "top")).not.toBeNull();
    expect(slot(container, "left")).toBeNull();
    expect(slot(container, "right")).toBeNull();
    expect(slot(container, "bottom")).toBeNull();
  });
});

describe("Grid rotate control", () => {
  it("should not render the rotate button when editable is false", () => {
    const { container } = render(
      <GridHost editable={false} nodeKey={NODE_KEY}>
        <div>body</div>
      </GridHost>,
    );
    expect(rotateButton(container)).toBeNull();
  });

  it("should render the rotate button when editable is true", () => {
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY}>
        <div>body</div>
      </GridHost>,
    );
    expect(rotateButton(container)).not.toBeNull();
  });

  it("should hide the rotate button when allowRotate is false", () => {
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY} allowRotate={false}>
        <div>body</div>
      </GridHost>,
    );
    expect(rotateButton(container)).toBeNull();
  });

  it.each<[location.Outer, location.Outer]>([
    ["left", "bottom"],
    ["bottom", "right"],
    ["right", "top"],
    ["top", "left"],
  ])(
    "should rotate orientation %s to %s clockwise on click",
    (orientation, expected) => {
      const onRotate = vi.fn();
      const { container } = render(
        <GridHost
          editable
          nodeKey={NODE_KEY}
          orientation={orientation}
          onRotate={onRotate}
        >
          <div>body</div>
        </GridHost>,
      );
      fireEvent.click(rotateButton(container) as HTMLElement);
      expect(onRotate).toHaveBeenCalledTimes(1);
      expect(onRotate).toHaveBeenCalledWith({ orientation: expected });
    },
  );
});

describe("Grid editable transitions", () => {
  // Renders Grid with an internal editable toggle so we can observe the
  // before/after side effect without the test having to remount the tree.
  const Toggle = ({ children }: PropsWithChildren) => {
    const [editable, setEditable] = useState(false);
    return (
      <Haul.Provider>
        <div className="react-flow__pane">
          <div data-id={NODE_KEY}>
            <button data-testid="toggle" onClick={() => setEditable((v) => !v)}>
              flip
            </button>
            <Grid.Grid editable={editable} nodeKey={NODE_KEY}>
              {children}
            </Grid.Grid>
          </div>
        </div>
      </Haul.Provider>
    );
  };

  it("should make items draggable on enter and not on exit of editable mode", () => {
    const { getByTestId } = render(
      <Toggle>
        <Grid.Item itemKey="x" location="top">
          <div data-testid="x">X</div>
        </Grid.Item>
      </Toggle>,
    );
    // Initially not editable: the item is rendered through StaticSlot, so
    // draggable is not set on the inner element.
    expect(getByTestId("x").getAttribute("draggable")).toBeNull();
    fireEvent.click(getByTestId("toggle"));
    // Now editable: cloneElement injects draggable="true".
    expect(getByTestId("x").getAttribute("draggable")).toBe("true");
    fireEvent.click(getByTestId("toggle"));
    expect(getByTestId("x").getAttribute("draggable")).toBeNull();
  });

  it("should not throw when the parent pane is missing from the ancestry", () => {
    // selectNode must succeed (so we wrap in data-id), but .react-flow__pane
    // is intentionally absent: reflowPane should silently skip in that case.
    const { getByTestId, rerender } = render(
      <Haul.Provider>
        <div data-id={NODE_KEY}>
          <Grid.Grid editable={false} nodeKey={NODE_KEY}>
            <span data-testid="body">body</span>
          </Grid.Grid>
        </div>
      </Haul.Provider>,
    );
    expect(getByTestId("body")).toBeTruthy();
    expect(() =>
      rerender(
        <Haul.Provider>
          <div data-id={NODE_KEY}>
            <Grid.Grid editable nodeKey={NODE_KEY}>
              <span data-testid="body">body</span>
            </Grid.Grid>
          </div>
        </Haul.Provider>,
      ),
    ).not.toThrow();
  });
});

describe("Grid drag-and-drop", () => {
  // Helper that gets the draggable inner element (the cloned child) for an
  // item whose data-testid is `itemTestId`. The cloneElement wraps the inner
  // div with draggable=true and the onDragStart handler.
  const draggable = (container: HTMLElement, itemTestId: string): HTMLElement =>
    container.querySelector(`[data-testid="${itemTestId}"]`) as HTMLElement;

  it("should set draggable=true and a grab cursor on items in editable mode", () => {
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY}>
        <Grid.Item itemKey="x" location="top">
          <div data-testid="x">X</div>
        </Grid.Item>
      </GridHost>,
    );
    const el = draggable(container, "x");
    expect(el.getAttribute("draggable")).toBe("true");
    expect(el.style.cursor).toBe("grab");
  });

  it("should call the dragged item's onLocationChange with the slot it is dragged over", () => {
    const onMoveX = vi.fn();
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY}>
        <Grid.Item itemKey="x" location="top" onLocationChange={onMoveX}>
          <div data-testid="x">X</div>
        </Grid.Item>
        <Grid.Item itemKey="y" location="right">
          <div data-testid="y">Y</div>
        </Grid.Item>
      </GridHost>,
    );
    fireEvent.dragStart(draggable(container, "x"));
    fireEvent.dragOver(slot(container, "right") as HTMLElement, {
      screenX: 100,
      screenY: 50,
    });
    expect(onMoveX).toHaveBeenCalledTimes(1);
    expect(onMoveX).toHaveBeenCalledWith("right");
  });

  it("should track movement across multiple slots in sequence", () => {
    const onMoveX = vi.fn();
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY}>
        <Grid.Item itemKey="x" location="top" onLocationChange={onMoveX}>
          <div data-testid="x">X</div>
        </Grid.Item>
        <Grid.Item itemKey="anchor-r" location="right">
          <div data-testid="anchor-r">R</div>
        </Grid.Item>
        <Grid.Item itemKey="anchor-b" location="bottom">
          <div data-testid="anchor-b">B</div>
        </Grid.Item>
      </GridHost>,
    );
    fireEvent.dragStart(draggable(container, "x"));
    fireEvent.dragOver(slot(container, "right") as HTMLElement, {
      screenX: 1,
      screenY: 1,
    });
    fireEvent.dragOver(slot(container, "bottom") as HTMLElement, {
      screenX: 2,
      screenY: 2,
    });
    expect(onMoveX.mock.calls.map(([loc]) => loc)).toEqual(["right", "bottom"]);
  });

  it("should not call onLocationChange before a drag has started", () => {
    const onMoveX = vi.fn();
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY}>
        <Grid.Item itemKey="x" location="top" onLocationChange={onMoveX}>
          <div data-testid="x">X</div>
        </Grid.Item>
        <Grid.Item itemKey="anchor" location="right">
          <div data-testid="anchor">A</div>
        </Grid.Item>
      </GridHost>,
    );
    // No dragStart — the haul state is empty, so canDrop returns false and
    // onDragOver should be a no-op.
    fireEvent.dragOver(slot(container, "right") as HTMLElement, {
      screenX: 5,
      screenY: 5,
    });
    expect(onMoveX).not.toHaveBeenCalled();
  });

  it("should ignore drag-over events that do not move the cursor", () => {
    const onMoveX = vi.fn();
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY}>
        <Grid.Item itemKey="x" location="top" onLocationChange={onMoveX}>
          <div data-testid="x">X</div>
        </Grid.Item>
        <Grid.Item itemKey="anchor" location="right">
          <div data-testid="anchor">A</div>
        </Grid.Item>
      </GridHost>,
    );
    fireEvent.dragStart(draggable(container, "x"));
    const right = slot(container, "right") as HTMLElement;
    fireEvent.dragOver(right, { screenX: 10, screenY: 10 });
    fireEvent.dragOver(right, { screenX: 10, screenY: 10 });
    // Same cursor coordinates short-circuit Haul's onDragOver.
    expect(onMoveX).toHaveBeenCalledTimes(1);
  });

  it("should reset Haul state on drop so a subsequent drag-over does not re-trigger", () => {
    const onMoveX = vi.fn();
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY}>
        <Grid.Item itemKey="x" location="top" onLocationChange={onMoveX}>
          <div data-testid="x">X</div>
        </Grid.Item>
        <Grid.Item itemKey="anchor" location="right">
          <div data-testid="anchor">A</div>
        </Grid.Item>
      </GridHost>,
    );
    fireEvent.dragStart(draggable(container, "x"));
    const right = slot(container, "right") as HTMLElement;
    fireEvent.dragOver(right, { screenX: 10, screenY: 10 });
    fireEvent.drop(right);
    onMoveX.mockClear();
    // Without another dragStart, the dragging state should be cleared and
    // no further onLocationChange should fire.
    fireEvent.dragOver(right, { screenX: 99, screenY: 99 });
    expect(onMoveX).not.toHaveBeenCalled();
  });

  it("should expose dragging visuals while a drag is in progress", () => {
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY}>
        <Grid.Item itemKey="x" location="top">
          <div data-testid="x">X</div>
        </Grid.Item>
        <Grid.Item itemKey="anchor" location="right">
          <div data-testid="anchor">A</div>
        </Grid.Item>
      </GridHost>,
    );
    const top = slot(container, "top") as HTMLElement;
    const right = slot(container, "right") as HTMLElement;
    expect(top.className).not.toContain("pluto-dragging");
    expect(right.className).not.toContain("pluto-dragging");
    fireEvent.dragStart(draggable(container, "x"));
    expect(top.className).toContain("pluto-dragging");
    expect(right.className).toContain("pluto-dragging");
    expect(top.className).toContain("pluto-haul-drop-region");
  });

  it("should mark the hovered slot with the dragging-over class and clear it on dragLeave", () => {
    const { container } = render(
      <GridHost editable nodeKey={NODE_KEY}>
        <Grid.Item itemKey="x" location="top">
          <div data-testid="x">X</div>
        </Grid.Item>
        <Grid.Item itemKey="anchor" location="right">
          <div data-testid="anchor">A</div>
        </Grid.Item>
      </GridHost>,
    );
    fireEvent.dragStart(draggable(container, "x"));
    const right = slot(container, "right") as HTMLElement;
    fireEvent.dragOver(right, { screenX: 1, screenY: 1 });
    expect(right.className).toContain("pluto-dragging-over");
    fireEvent.dragLeave(right);
    expect(right.className).not.toContain("pluto-dragging-over");
  });

  it("should not register dragstart handlers on items in static (non-editable) mode", () => {
    const onMoveX = vi.fn();
    const { container } = render(
      <GridHost editable={false} nodeKey={NODE_KEY}>
        <Grid.Item itemKey="x" location="top" onLocationChange={onMoveX}>
          <div data-testid="x">X</div>
        </Grid.Item>
        <Grid.Item itemKey="anchor" location="right">
          <div data-testid="anchor">A</div>
        </Grid.Item>
      </GridHost>,
    );
    fireEvent.dragStart(draggable(container, "x"));
    fireEvent.dragOver(slot(container, "right") as HTMLElement, {
      screenX: 10,
      screenY: 10,
    });
    expect(onMoveX).not.toHaveBeenCalled();
  });
});

describe("Grid resize controls", () => {
  const EDGES = ["top", "right", "bottom", "left"];
  const CORNERS = ["top-left", "top-right", "bottom-left", "bottom-right"];

  const ResizableHost = ({
    editable = true,
    onResize,
    keepAspectRatio,
  }: Partial<Grid.GridProps>): ReactElement => (
    <ReactFlowProvider>
      <Haul.Provider>
        <DiagramContext value={diagramCtx}>
          <div data-id={NODE_KEY}>
            <Grid.Grid
              editable={editable}
              nodeKey={NODE_KEY}
              onResize={onResize}
              keepAspectRatio={keepAspectRatio}
            >
              <div>body</div>
            </Grid.Grid>
          </div>
        </DiagramContext>
      </Haul.Provider>
    </ReactFlowProvider>
  );

  const controlEls = (c: HTMLElement) =>
    c.querySelectorAll<HTMLElement>(".react-flow__resize-control");
  const hasControl = (c: HTMLElement, ...classes: string[]) =>
    Array.from(controlEls(c)).some((el) =>
      classes.every((cl) => el.classList.contains(cl)),
    );

  beforeEach(() => resizeControls.clear());

  describe("render gating", () => {
    it("should render no controls when the node is not editable", () => {
      const { container } = render(
        <ResizableHost editable={false} onResize={vi.fn()} />,
      );
      expect(controlEls(container)).toHaveLength(0);
    });

    it("should render no controls when no onResize handler is provided", () => {
      const { container } = render(<ResizableHost />);
      expect(controlEls(container)).toHaveLength(0);
    });

    it("should render eight controls when editable and resizable", () => {
      const { container } = render(<ResizableHost onResize={vi.fn()} />);
      expect(controlEls(container)).toHaveLength(8);
    });
  });

  describe("control rendering", () => {
    it("should render each edge as a line-variant control at its position", () => {
      const { container } = render(<ResizableHost onResize={vi.fn()} />);
      EDGES.forEach((edge) =>
        expect(hasControl(container, edge, ResizeControlVariant.Line)).toBe(true),
      );
    });

    it("should render each corner as a handle-variant control at its position", () => {
      const { container } = render(<ResizableHost onResize={vi.fn()} />);
      CORNERS.forEach((corner) => {
        const [vertical, horizontal] = corner.split("-");
        expect(
          hasControl(container, vertical, horizontal, ResizeControlVariant.Handle),
        ).toBe(true);
      });
    });
  });

  describe("aspect ratio", () => {
    it("should lock the corners but not the edges by default", () => {
      render(<ResizableHost onResize={vi.fn()} />);
      CORNERS.forEach((corner) =>
        expect(resizeControls.get(corner)?.keepAspectRatio).toBe(true),
      );
      EDGES.forEach((edge) =>
        expect(resizeControls.get(edge)?.keepAspectRatio).toBeFalsy(),
      );
    });

    it("should lock every control when keepAspectRatio is set", () => {
      render(<ResizableHost keepAspectRatio onResize={vi.fn()} />);
      [...EDGES, ...CORNERS].forEach((position) =>
        expect(resizeControls.get(position)?.keepAspectRatio).toBe(true),
      );
    });
  });

  describe("dimension forwarding", () => {
    it("should round fractional dimensions before forwarding them", () => {
      const onResize = vi.fn();
      render(<ResizableHost onResize={onResize} />);
      resizeControls.get("right")?.triggerResize(120.6, 80.4);
      expect(onResize).toHaveBeenCalledTimes(1);
      expect(onResize).toHaveBeenCalledWith({ width: 121, height: 80 });
    });

    it("should forward rounded dimensions from a corner control", () => {
      const onResize = vi.fn();
      render(<ResizableHost onResize={onResize} />);
      resizeControls.get("bottom-right")?.triggerResize(50.5, 50.49);
      expect(onResize).toHaveBeenCalledWith({ width: 51, height: 50 });
    });
  });
});
