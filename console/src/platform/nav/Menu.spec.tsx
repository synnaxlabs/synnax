// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { act, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Nav } from "@/platform/nav";
import { renderWithConsole } from "@/testutil";

const item = (key: string, overrides: Partial<Nav.Item> = {}): Nav.Item => ({
  key,
  content: <div />,
  icon: <Icon.Add aria-label={`icon-${key}`} />,
  tooltip: key,
  trigger: [],
  ...overrides,
});

const clickItem = (key: string): void => {
  const icon = screen.getByLabelText(`icon-${key}`);
  const clickable = icon.closest("button") ?? icon.parentElement;
  if (clickable == null) throw new Error(`no clickable ancestor for ${key}`);
  fireEvent.click(clickable);
};

const noop = {
  onStartHover: vi.fn(),
  onStopHover: vi.fn(),
  onToggle: vi.fn(),
  onPin: vi.fn(),
};

describe("nav Menu", () => {
  it("should render every item's icon", async () => {
    await renderWithConsole(
      <Nav.Menu items={[item("alpha"), item("bravo")]} onSelect={vi.fn()} {...noop} />,
    );
    expect(screen.getByLabelText("icon-alpha")).toBeTruthy();
    expect(screen.getByLabelText("icon-bravo")).toBeTruthy();
  });

  it("should not render items whose useVisible returns false", async () => {
    await renderWithConsole(
      <Nav.Menu
        items={[item("alpha"), item("bravo", { useVisible: () => false })]}
        onSelect={vi.fn()}
        {...noop}
      />,
    );
    expect(screen.getByLabelText("icon-alpha")).toBeTruthy();
    expect(screen.queryByLabelText("icon-bravo")).toBeNull();
  });

  it("should call onSelect with the item key when an item is clicked", async () => {
    const onSelect = vi.fn();
    await renderWithConsole(
      <Nav.Menu items={item("alpha")} onSelect={onSelect} {...noop} />,
    );
    clickItem("alpha");
    expect(onSelect).toHaveBeenCalledWith("alpha");
  });

  describe("hover", () => {
    afterEach(() => {
      vi.useRealTimers();
    });

    it("should start hovering after the enter delay and stop when the pointer drifts away", async () => {
      const onStartHover = vi.fn();
      const onStopHover = vi.fn();
      await renderWithConsole(
        <Nav.Menu
          items={item("alpha")}
          onSelect={vi.fn()}
          onStartHover={onStartHover}
          onStopHover={onStopHover}
          onToggle={vi.fn()}
          onPin={vi.fn()}
        />,
      );
      const icon = screen.getByLabelText("icon-alpha");
      const menuItem = icon.closest("button") ?? icon.parentElement!;
      vi.useFakeTimers();
      fireEvent.mouseEnter(menuItem, { clientX: 0, clientY: 0 });
      act(() => {
        vi.advanceTimersByTime(350);
      });
      expect(onStartHover).toHaveBeenCalledWith("alpha");
      act(() => {
        window.dispatchEvent(new MouseEvent("mousemove", { clientX: 0, clientY: 200 }));
      });
      expect(onStopHover).toHaveBeenCalledTimes(1);
    });
  });
});
