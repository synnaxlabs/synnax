// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { act, fireEvent, screen } from "@testing-library/react";
import {
  type PropsWithChildren,
  type ReactElement,
  Suspense,
  use,
  useLayoutEffect,
} from "react";
import { describe, expect, it, vi } from "vitest";

import { Nav } from "@/platform/nav";
import { Session } from "@/session";
import { createTestStore, renderWithConsole, type TestStore } from "@/testutil";

const ITEM_KEY = "alpha";
const CONTENT = "alpha content";
const SIBLING = "sibling";
const DRAWER_SELECTOR = ".console-nav__drawer";

const Harness = ({ children }: PropsWithChildren): ReactElement => {
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
      {children}
    </Nav.Drawer>
  );
};
Harness.displayName = "DrawerHarness";

interface SiblingProps {
  onMount: () => void;
}

const Sibling = ({ onMount }: SiblingProps): ReactElement => {
  useLayoutEffect(onMount, [onMount]);
  return <div>{SIBLING}</div>;
};

interface PendingProps {
  content: Promise<string>;
}

const Pending = ({ content }: PendingProps): ReactElement => <div>{use(content)}</div>;

const renderDrawer = async (): Promise<{
  store: TestStore;
  container: HTMLElement;
}> => {
  const store = await createTestStore();
  const { container } = await renderWithConsole(
    <Harness>
      <div>{CONTENT}</div>
    </Harness>,
    { store },
  );
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
  store.getState()[Session.Nav.SLICE_NAME].windows[MAIN_WINDOW]?.left;

describe("Nav.Drawer", () => {
  it("should show content when an item is selected and hide it when toggled off", async () => {
    const { store } = await renderDrawer();
    expect(screen.queryByText(CONTENT)).toBeNull();
    selectItem(store);
    expect(screen.getByText(CONTENT)).toBeTruthy();
    expect(leftState(store)?.selected).toBe(ITEM_KEY);
    selectItem(store);
    expect(screen.queryByText(CONTENT)).toBeNull();
    expect(leftState(store)?.selected).toBeUndefined();
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
    expect(leftState(store)?.selected).toBeUndefined();
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

  it("should not restart content beside the drawer while its content waits for data", async () => {
    const onMount = vi.fn();
    let resolve: (content: string) => void = () => {};
    const content = new Promise<string>((r) => (resolve = r));
    const store = await createTestStore();
    // Stands in for the boundary that wraps the whole workspace.
    await renderWithConsole(
      <Suspense fallback={null}>
        <Sibling onMount={onMount} />
        <Harness>
          <Pending content={content} />
        </Harness>
      </Suspense>,
      { store },
    );
    await act(async () => {
      store.dispatch(Session.Nav.selectLeft({ key: ITEM_KEY }));
    });
    await act(async () => {
      resolve(CONTENT);
      await content;
    });
    expect(screen.getByText(CONTENT)).toBeTruthy();
    expect(onMount).toHaveBeenCalledTimes(1);
  });
});
