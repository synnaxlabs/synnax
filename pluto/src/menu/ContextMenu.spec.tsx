// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";

import { Menu } from "@/menu";

const TestMenu = (): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu
      menu={({ keys }) => (
        <Menu.Menu>
          <Menu.Item itemKey="action1">Action 1</Menu.Item>
          <Menu.Item itemKey="action2">Action 2</Menu.Item>
          <span data-testid="menu-keys">{keys.join(",")}</span>
        </Menu.Menu>
      )}
      {...ctx}
    >
      <div
        data-testid="target"
        id="target-1"
        className="pluto-context-target"
        onContextMenu={ctx.open}
      >
        Right click me
      </div>
    </Menu.ContextMenu>
  );
};

describe("ContextMenu", () => {
  it("should not display the menu by default", () => {
    render(<TestMenu />);
    expect(screen.queryByText("Action 1")).toBeNull();
  });

  it("should display the menu on context menu event", () => {
    render(<TestMenu />);
    fireEvent.contextMenu(screen.getByTestId("target"));
    expect(screen.getByText("Action 1")).toBeTruthy();
    expect(screen.getByText("Action 2")).toBeTruthy();
  });

  it("should close the menu when clicking a menu item", () => {
    render(<TestMenu />);
    fireEvent.contextMenu(screen.getByTestId("target"));
    expect(screen.getByText("Action 1")).toBeTruthy();
    fireEvent.click(screen.getByText("Action 1"));
    expect(screen.queryByText("Action 1")).toBeNull();
  });

  it("should provide selected keys from context targets", () => {
    render(<TestMenu />);
    fireEvent.contextMenu(screen.getByTestId("target"));
    expect(screen.getByTestId("menu-keys").textContent).toBe("target-1");
  });

  it("should recalculate position via Dialog.position on window resize", () => {
    const spy = vi.spyOn(Dialog, "position");
    render(<TestMenu />);
    fireEvent.contextMenu(screen.getByTestId("target"));
    const callsBefore = spy.mock.calls.length;
    act(() => {
      fireEvent(window, new Event("resize"));
    });
    expect(spy.mock.calls.length).toBeGreaterThan(callsBefore);
    spy.mockRestore();
  });
});
