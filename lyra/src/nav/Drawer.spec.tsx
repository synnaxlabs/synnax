// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";
import { act, fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Nav } from "@/nav";

const renderDrawer = (
  props: Partial<Nav.DrawerProps> = {},
): ReturnType<typeof render> =>
  render(
    <Nav.Drawer location="left" size={200} {...props}>
      <p>Hello</p>
    </Nav.Drawer>,
  );

const drawerOf = (c: ReturnType<typeof render>): HTMLElement => {
  const drawer = c.container.querySelector<HTMLElement>(".pluto-nav-drawer");
  if (drawer == null) throw new Error("nav drawer not found");
  return drawer;
};

const handleOf = (c: ReturnType<typeof render>): HTMLElement => {
  const handle = c.container.querySelector<HTMLElement>(".pluto-resize__handle");
  if (handle == null) throw new Error("resize handle not found");
  return handle;
};

// Cursor.useDrag captures the pointer on the handle, then tracks moves/up on window.
// The from -> to distance must exceed the activation threshold for the drag to begin.
// Moves are coalesced onto the next animation frame, so pressMove awaits one.
const pressMove = async (
  c: ReturnType<typeof render>,
  from: xy.XY,
  to: xy.XY,
): Promise<void> => {
  fireEvent.pointerDown(handleOf(c), {
    pointerId: 1,
    button: 0,
    isPrimary: true,
    clientX: from.x,
    clientY: from.y,
  });
  fireEvent.pointerMove(window, { pointerId: 1, clientX: to.x, clientY: to.y });
  await act(
    async () =>
      await new Promise<void>((resolve) => requestAnimationFrame(() => resolve())),
  );
};

const release = (to: xy.XY): void => {
  fireEvent.pointerUp(window, { pointerId: 1, clientX: to.x, clientY: to.y });
};

const drag = async (
  c: ReturnType<typeof render>,
  from: xy.XY,
  to: xy.XY,
): Promise<void> => {
  await pressMove(c, from, to);
  release(to);
};

describe("Nav.Drawer", () => {
  it("should render its children", () => {
    const c = renderDrawer();
    expect(c.getByText("Hello")).toBeTruthy();
  });

  it("should render a resize handle", () => {
    const c = renderDrawer();
    expect(handleOf(c)).toBeTruthy();
  });

  it("should forward an arbitrary class name", () => {
    const c = renderDrawer({ className: "custom-drawer" });
    expect(drawerOf(c).className).toContain("custom-drawer");
  });

  describe("visibility", () => {
    it("should be visible when not collapsed", () => {
      const c = renderDrawer({ collapsed: false });
      const drawer = drawerOf(c);
      expect(drawer.className).toContain("pluto--visible");
      expect(drawer.className).not.toContain("pluto--hidden");
    });

    it("should be hidden when collapsed", () => {
      const c = renderDrawer({ collapsed: true });
      const drawer = drawerOf(c);
      expect(drawer.className).toContain("pluto--hidden");
      expect(drawer.className).not.toContain("pluto--visible");
    });
  });

  describe("resize within the collapse threshold", () => {
    it("should call onResize with the clamped size", async () => {
      const onResize = vi.fn();
      const c = renderDrawer({ onResize });
      await drag(c, { x: 500, y: 0 }, { x: 460, y: 0 });
      expect(onResize).toHaveBeenCalled();
      expect(onResize.mock.lastCall?.[0]).toEqual(160);
    });

    it("should call onResizeEnd and not onCollapse on release", async () => {
      const onResizeEnd = vi.fn();
      const onCollapse = vi.fn();
      const c = renderDrawer({ onResizeEnd, onCollapse });
      await drag(c, { x: 500, y: 0 }, { x: 460, y: 0 });
      expect(onResizeEnd).toHaveBeenCalledWith(160, expect.anything());
      expect(onCollapse).not.toHaveBeenCalled();
    });

    it("should stay visible throughout the drag", async () => {
      const c = renderDrawer();
      await pressMove(c, { x: 500, y: 0 }, { x: 460, y: 0 });
      expect(drawerOf(c).className).toContain("pluto--visible");
      release({ x: 460, y: 0 });
    });
  });

  describe("resize past the collapse threshold", () => {
    it("should hide the drawer mid-drag and suppress onResize", async () => {
      const onResize = vi.fn();
      const c = renderDrawer({ onResize });
      await pressMove(c, { x: 500, y: 0 }, { x: 250, y: 0 });
      expect(drawerOf(c).className).toContain("pluto--hidden");
      expect(onResize).not.toHaveBeenCalled();
      release({ x: 250, y: 0 });
    });

    it("should call onCollapse and not onResizeEnd on release", async () => {
      const onResizeEnd = vi.fn();
      const onCollapse = vi.fn();
      const c = renderDrawer({ onResizeEnd, onCollapse });
      await drag(c, { x: 500, y: 0 }, { x: 250, y: 0 });
      expect(onCollapse).toHaveBeenCalledTimes(1);
      expect(onResizeEnd).not.toHaveBeenCalled();
    });

    it("should clear the transient collapsed state after release", async () => {
      const c = renderDrawer();
      await drag(c, { x: 500, y: 0 }, { x: 250, y: 0 });
      expect(drawerOf(c).className).toContain("pluto--visible");
    });
  });

  describe("collapseThreshold", () => {
    it("should collapse for a smaller overshoot when the threshold is low", async () => {
      const onCollapse = vi.fn();
      const c = renderDrawer({ collapseThreshold: 10, onCollapse });
      await drag(c, { x: 500, y: 0 }, { x: 380, y: 0 });
      expect(onCollapse).toHaveBeenCalledTimes(1);
    });

    it("should not collapse for the same overshoot when the threshold is high", async () => {
      const onCollapse = vi.fn();
      const onResizeEnd = vi.fn();
      const c = renderDrawer({ onCollapse, onResizeEnd });
      await drag(c, { x: 500, y: 0 }, { x: 380, y: 0 });
      expect(onCollapse).not.toHaveBeenCalled();
      expect(onResizeEnd).toHaveBeenCalled();
    });
  });

  describe("when already collapsed", () => {
    it("should ignore drags entirely", async () => {
      const onResize = vi.fn();
      const onResizeEnd = vi.fn();
      const onCollapse = vi.fn();
      const c = renderDrawer({ collapsed: true, onResize, onResizeEnd, onCollapse });
      await drag(c, { x: 500, y: 0 }, { x: 250, y: 0 });
      expect(onResize).not.toHaveBeenCalled();
      expect(onResizeEnd).not.toHaveBeenCalled();
      expect(onCollapse).not.toHaveBeenCalled();
    });
  });
});
