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

import { OS } from "@/os";

const WINDOWS_CONTROL = ".pluto-windows-control";
const MACOS_CONTROL = ".pluto-macos-control";

const control = (c: HTMLElement, os: "windows" | "macos", action: string) =>
  c.querySelector<HTMLButtonElement>(`.pluto-${os}-control--${action}`);

describe("OS.Controls", () => {
  describe("os dispatch", () => {
    it("should render the windows controls when forced to Windows", () => {
      const { container } = render(<OS.Controls forceOS="Windows" />);
      expect(container.querySelectorAll(WINDOWS_CONTROL)).toHaveLength(3);
      expect(container.querySelectorAll(MACOS_CONTROL)).toHaveLength(0);
    });

    it("should render the macOS controls when forced to macOS", () => {
      const { container } = render(<OS.Controls forceOS="macOS" />);
      expect(container.querySelectorAll(MACOS_CONTROL)).toHaveLength(3);
      expect(container.querySelectorAll(WINDOWS_CONTROL)).toHaveLength(0);
    });

    it("should render nothing when visibleIfOS does not match the os", () => {
      const { container } = render(
        <OS.Controls forceOS="Windows" visibleIfOS="macOS" />,
      );
      expect(container.querySelector("button")).toBeNull();
    });

    it("should render when visibleIfOS matches the os", () => {
      const { container } = render(<OS.Controls forceOS="macOS" visibleIfOS="macOS" />);
      expect(container.querySelectorAll(MACOS_CONTROL)).toHaveLength(3);
    });
  });

  describe("windows", () => {
    it("should show the box glyph when the window is not maximized", () => {
      const { container } = render(<OS.Controls forceOS="Windows" />);
      const maximize = control(container, "windows", "maximize");
      expect(maximize?.querySelector(".pluto-icon--box")).not.toBeNull();
      expect(maximize?.querySelector(".pluto-icon--boxes")).toBeNull();
    });

    it("should swap in the boxes glyph when the window is maximized", () => {
      const { container } = render(<OS.Controls forceOS="Windows" maximized />);
      const maximize = control(container, "windows", "maximize");
      expect(maximize?.querySelector(".pluto-icon--boxes")).not.toBeNull();
      expect(maximize?.querySelector(".pluto-icon--box")).toBeNull();
    });

    it("should close with the thin glyph rather than the heavier Icon.Close", () => {
      const { container } = render(<OS.Controls forceOS="Windows" />);
      const close = control(container, "windows", "close");
      expect(close?.querySelector(".pluto-icon--close-thin")).not.toBeNull();
      expect(close?.querySelector(".pluto-icon--close")).toBeNull();
    });

    it("should call the handler matching each control", () => {
      const onMinimize = vi.fn();
      const onMaximize = vi.fn();
      const onClose = vi.fn();
      const { container } = render(
        <OS.Controls
          forceOS="Windows"
          onClose={onClose}
          onMaximize={onMaximize}
          onMinimize={onMinimize}
        />,
      );
      fireEvent.click(control(container, "windows", "minimize") as HTMLElement);
      fireEvent.click(control(container, "windows", "maximize") as HTMLElement);
      fireEvent.click(control(container, "windows", "close") as HTMLElement);
      expect(onMinimize).toHaveBeenCalledTimes(1);
      expect(onMaximize).toHaveBeenCalledTimes(1);
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it("should omit a disabled control instead of rendering it inert", () => {
      const { container } = render(
        <OS.Controls forceOS="Windows" disabled={["maximize"]} />,
      );
      expect(control(container, "windows", "maximize")).toBeNull();
      expect(container.querySelectorAll(WINDOWS_CONTROL)).toHaveLength(2);
    });
  });

  describe("macOS", () => {
    it("should fullscreen from the maximize light rather than maximize", () => {
      const onMaximize = vi.fn();
      const onFullscreen = vi.fn();
      const { container } = render(
        <OS.Controls
          forceOS="macOS"
          onFullscreen={onFullscreen}
          onMaximize={onMaximize}
        />,
      );
      fireEvent.click(control(container, "macos", "maximize") as HTMLElement);
      expect(onFullscreen).toHaveBeenCalledTimes(1);
      expect(onMaximize).not.toHaveBeenCalled();
    });

    it("should call the handler matching each remaining control", () => {
      const onMinimize = vi.fn();
      const onClose = vi.fn();
      const { container } = render(
        <OS.Controls forceOS="macOS" onClose={onClose} onMinimize={onMinimize} />,
      );
      fireEvent.click(control(container, "macos", "minimize") as HTMLElement);
      fireEvent.click(control(container, "macos", "close") as HTMLElement);
      expect(onMinimize).toHaveBeenCalledTimes(1);
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it("should keep a disabled control mounted but inert", () => {
      const onClose = vi.fn();
      const { container } = render(
        <OS.Controls forceOS="macOS" disabled={["close"]} onClose={onClose} />,
      );
      const close = control(container, "macos", "close");
      expect(close).not.toBeNull();
      expect(close?.disabled).toBe(true);
      fireEvent.click(close as HTMLElement);
      expect(onClose).not.toHaveBeenCalled();
    });

    it("should mark the controls blurred when the window is not focused", () => {
      const { container } = render(<OS.Controls focused={false} forceOS="macOS" />);
      expect(container.querySelector(".pluto-macos-controls--blurred")).not.toBeNull();
    });

    it("should not mark the controls blurred when the window is focused", () => {
      const { container } = render(<OS.Controls focused forceOS="macOS" />);
      expect(container.querySelector(".pluto-macos-controls--blurred")).toBeNull();
    });
  });
});
