// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen } from "@testing-library/react";
import { act, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Nav } from "@/platform/nav";
import { Session } from "@/session";
import { createTestStore, renderWithConsole, type TestStore } from "@/testutil";

const ITEM_KEY = "alpha";
const CONTENT = "alpha content";
const DRAWER_SELECTOR = ".console-nav__drawer";

const Harness = (): ReactElement => {
  const { selected, hover, size } = Session.Nav.useSelectLeft();
  const dispatch = Session.useDispatch();
  return (
    <Nav.Drawer
      location="left"
      size={size ?? 256}
      hover={hover}
      collapsed={selected !== ITEM_KEY}
      onResizeEnd={(next) => dispatch(Session.Nav.resizeLeft({ size: next }))}
      onCollapse={() => dispatch(Session.Nav.collapseLeft({}))}
      onStopHover={() => dispatch(Session.Nav.stopLeftHover({}))}
    >
      <div>{CONTENT}</div>
    </Nav.Drawer>
  );
};
Harness.displayName = "DrawerHarness";

const renderDrawer = async (): Promise<{
  store: TestStore;
  container: HTMLElement;
}> => {
  const store = await createTestStore();
  const { container } = await renderWithConsole(<Harness />, { store });
  return { store, container };
};

const selectItem = (store: TestStore): void =>
  act(() => {
    store.dispatch(Session.Nav.selectLeft({ key: ITEM_KEY }));
  });

const startHover = (store: TestStore): void =>
  act(() => {
    store.dispatch(Session.Nav.startLeftHover({ key: ITEM_KEY }));
  });

const moveMouse = (init: MouseEventInit): void =>
  act(() => {
    window.dispatchEvent(new MouseEvent("mousemove", init));
  });

const leftState = (store: TestStore) =>
  Session.Nav.selectWindowState(store.getState()).left;

describe("Nav.Drawer", () => {
  it("should show content when an item is selected and hide it when toggled off", async () => {
    const { store } = await renderDrawer();
    expect(screen.queryByText(CONTENT)).toBeNull();
    selectItem(store);
    expect(screen.getByText(CONTENT)).toBeTruthy();
    expect(Session.Nav.selectLeftSelected(store.getState())).toBe(ITEM_KEY);
    selectItem(store);
    expect(screen.queryByText(CONTENT)).toBeNull();
    expect(Session.Nav.selectLeftSelected(store.getState())).toBeUndefined();
  });

  it("should size the drawer from the nav state after a resize action", async () => {
    const { store, container } = await renderDrawer();
    selectItem(store);
    act(() => {
      store.dispatch(Session.Nav.resizeLeft({ size: 312 }));
    });
    expect(leftState(store).size).toBe(312);
    const drawer = container.querySelector<HTMLElement>(DRAWER_SELECTOR);
    expect(drawer).not.toBeNull();
    expect(drawer?.style.width).toBe("312px");
  });

  it("should stop hovering when the pointer leaves the drawer's threshold region", async () => {
    const { store, container } = await renderDrawer();
    startHover(store);
    expect(leftState(store).hover).toBe(true);
    expect(screen.getByText(CONTENT)).toBeTruthy();
    const drawer = container.querySelector<HTMLElement>(DRAWER_SELECTOR);
    expect(drawer).not.toBeNull();
    fireEvent.mouseLeave(drawer as HTMLElement);
    moveMouse({ clientX: 0, clientY: 0 });
    expect(leftState(store).hover).toBe(true);
    moveMouse({ clientX: 300, clientY: 300 });
    expect(leftState(store).hover).toBe(false);
    expect(Session.Nav.selectLeftSelected(store.getState())).toBeUndefined();
    expect(screen.queryByText(CONTENT)).toBeNull();
  });

  it("should keep hovering when the pointer moves away with a button held", async () => {
    const { store, container } = await renderDrawer();
    startHover(store);
    const drawer = container.querySelector<HTMLElement>(DRAWER_SELECTOR);
    fireEvent.mouseLeave(drawer as HTMLElement);
    moveMouse({ clientX: 0, clientY: 0, buttons: 1 });
    moveMouse({ clientX: 300, clientY: 300 });
    expect(leftState(store).hover).toBe(true);
    expect(screen.getByText(CONTENT)).toBeTruthy();
  });
});
