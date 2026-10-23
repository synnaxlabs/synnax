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
import { describe, expect, it } from "vitest";

import { Menu } from "@/menu";
import { Triggers } from "@/triggers";

const TestMenu = (): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu
      menu={() => (
        <Menu.Menu>
          <Menu.Item itemKey="action1">Action 1</Menu.Item>
          <Menu.Item itemKey="action2">Action 2</Menu.Item>
        </Menu.Menu>
      )}
      {...ctx}
    >
      <div id="target-1" className="pluto-context-target" onContextMenu={ctx.open}>
        Right click me
      </div>
    </Menu.ContextMenu>
  );
};

const StampMenu = (): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu
      menu={() => (
        <Menu.Menu>
          <Menu.Item itemKey="action1">Action 1</Menu.Item>
        </Menu.Menu>
      )}
      {...ctx}
    >
      <div>
        <div
          id="row-1"
          className={`${Menu.CONTEXT_TARGET} ${Menu.CONTEXT_SELECTED}`}
          onContextMenu={ctx.open}
        >
          Row 1
        </div>
        <div
          id="row-2"
          className={`${Menu.CONTEXT_TARGET} ${Menu.CONTEXT_SELECTED}`}
          onContextMenu={ctx.open}
        >
          Row 2
        </div>
        <div id="row-3" className={Menu.CONTEXT_TARGET} onContextMenu={ctx.open}>
          Row 3
        </div>
        <div id="row-4" className={Menu.CONTEXT_TARGET} onContextMenu={ctx.open}>
          Row 4
        </div>
      </div>
    </Menu.ContextMenu>
  );
};

const stamped = (text: string): boolean =>
  screen.getByText(text).hasAttribute(Menu.CONTEXT_OPEN_ATTRIBUTE);

const KeyMenu = ({ onKeys }: { onKeys: (keys: string[]) => void }): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu menu={({ keys }) => <>{onKeys(keys)}</>} {...ctx}>
      <div
        id="dom-id"
        data-menu-key="semantic-key"
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

  it("should key a context target by its data-menu-key over its id", () => {
    let captured: string[] = [];
    render(<KeyMenu onKeys={(keys) => (captured = keys)} />);
    fireEvent.contextMenu(screen.getByText("Right click me"));
    expect(captured).toEqual(["semantic-key"]);
  });

  it("should display the menu on context menu event", () => {
    render(<TestMenu />);
    fireEvent.contextMenu(screen.getByText("Right click me"));
    expect(screen.getByText("Action 1")).toBeTruthy();
    expect(screen.getByText("Action 2")).toBeTruthy();
  });

  it("should close the menu when clicking a menu item", () => {
    render(<TestMenu />);
    fireEvent.contextMenu(screen.getByText("Right click me"));
    expect(screen.getByText("Action 1")).toBeTruthy();
    fireEvent.click(screen.getByText("Action 1"));
    expect(screen.queryByText("Action 1")).toBeNull();
  });

  it("should keep the menu visible after a window resize", () => {
    render(<TestMenu />);
    fireEvent.contextMenu(screen.getByText("Right click me"));
    expect(screen.getByText("Action 1")).toBeTruthy();
    act(() => {
      fireEvent(window, new Event("resize"));
    });
    expect(screen.getByText("Action 1")).toBeTruthy();
    expect(screen.getByText("Action 2")).toBeTruthy();
  });

  describe("open target stamping", () => {
    it("should stamp the clicked target while the menu is open", () => {
      render(<StampMenu />);
      fireEvent.contextMenu(screen.getByText("Row 3"));
      expect(stamped("Row 3")).toBe(true);
      expect(stamped("Row 1")).toBe(false);
    });

    it("should stamp only the clicked target even when it is selected", () => {
      render(<StampMenu />);
      fireEvent.contextMenu(screen.getByText("Row 1"));
      expect(stamped("Row 1")).toBe(true);
      expect(stamped("Row 2")).toBe(false);
      expect(stamped("Row 3")).toBe(false);
    });

    it("should remove the stamp when the menu closes", () => {
      render(<StampMenu />);
      fireEvent.contextMenu(screen.getByText("Row 3"));
      fireEvent.click(screen.getByText("Action 1"));
      expect(stamped("Row 3")).toBe(false);
    });

    it("should move the stamp when another target is right clicked", () => {
      render(<StampMenu />);
      fireEvent.contextMenu(screen.getByText("Row 3"));
      fireEvent.contextMenu(screen.getByText("Row 4"));
      expect(stamped("Row 3")).toBe(false);
      expect(stamped("Row 4")).toBe(true);
    });

    it("should remove the stamp when Escape closes the menu", () => {
      render(
        <Triggers.Provider>
          <StampMenu />
        </Triggers.Provider>,
      );
      fireEvent.contextMenu(screen.getByText("Row 3"));
      expect(stamped("Row 3")).toBe(true);
      act(() => {
        fireEvent.keyDown(window, { key: "Escape", code: "Escape" });
      });
      expect(screen.queryByText("Action 1")).toBeNull();
      expect(stamped("Row 3")).toBe(false);
    });

    it("should not stamp a fallback element that is not a context target", () => {
      render(<TestMenu />);
      const el = screen.getByText("Right click me");
      fireEvent.contextMenu(el);
      expect(screen.getByText("Action 1")).toBeTruthy();
      expect(el.hasAttribute(Menu.CONTEXT_OPEN_ATTRIBUTE)).toBe(false);
    });
  });
});
