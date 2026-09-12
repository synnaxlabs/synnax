// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Triggers } from "@/triggers";
import { useTriggers, type UseTriggersProps } from "@/vis/diagram/useTriggers";

const noop = () => {};
const noopCursor = () => {};

const defaultProps: UseTriggersProps = {
  onUndo: noop,
  onRedo: noop,
  onCopy: noopCursor,
  onPaste: noopCursor,
  onClearSelection: noop,
  onSelectAll: noop,
};

const wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>{children}</Triggers.Provider>
);

const renderTriggers = (props: Partial<UseTriggersProps>): void => {
  renderHook(() => useTriggers({ ...defaultProps, ...props }), { wrapper });
};

describe("Diagram.useTriggers", () => {
  it("should call onUndo when Control+Z is pressed", () => {
    const onUndo = vi.fn();
    renderTriggers({ onUndo });
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyZ", ctrlKey: true });
    expect(onUndo).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyZ" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  it("should call onRedo when Control+Shift+Z is pressed", () => {
    const onRedo = vi.fn();
    renderTriggers({ onRedo });
    fireEvent.keyDown(document.body, { code: "ShiftLeft", shiftKey: true });
    fireEvent.keyDown(document.body, { code: "ControlLeft", shiftKey: true });
    fireEvent.keyDown(document.body, {
      code: "KeyZ",
      ctrlKey: true,
      shiftKey: true,
    });
    expect(onRedo).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyZ" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
    fireEvent.keyUp(document.body, { code: "ShiftLeft" });
  });

  it("should call onCopy when Control+C is pressed", () => {
    const onCopy = vi.fn();
    renderTriggers({ onCopy });
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyC", ctrlKey: true });
    expect(onCopy).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyC" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  it("should call onPaste when Control+V is pressed", () => {
    const onPaste = vi.fn();
    renderTriggers({ onPaste });
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyV", ctrlKey: true });
    expect(onPaste).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyV" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  it("should call onGroup when Control+G is pressed", () => {
    const onGroup = vi.fn();
    renderTriggers({ onGroup });
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyG", ctrlKey: true });
    expect(onGroup).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyG" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  it("should call onUngroup when Control+U is pressed", () => {
    const onUngroup = vi.fn();
    renderTriggers({ onUngroup });
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyU", ctrlKey: true });
    expect(onUngroup).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyU" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  it("should call onClear when Escape is pressed", () => {
    const onClearSelection = vi.fn();
    renderTriggers({ onClearSelection });
    fireEvent.keyDown(document.body, { code: "Escape" });
    expect(onClearSelection).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "Escape" });
  });

  it("should call onSelectAll when Control+A is pressed", () => {
    const onSelectAll = vi.fn();
    renderTriggers({ onSelectAll });
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyA", ctrlKey: true });
    expect(onSelectAll).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyA" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  describe("read-only diagrams", () => {
    const press = (...codes: string[]): void => {
      codes.forEach((code) => fireEvent.keyDown(document.body, { code }));
      [...codes].reverse().forEach((code) => fireEvent.keyUp(document.body, { code }));
    };

    it("should withhold the shortcuts that change the diagram", () => {
      const onUndo = vi.fn();
      const onRedo = vi.fn();
      const onPaste = vi.fn();
      const onGroup = vi.fn();
      const onUngroup = vi.fn();
      renderTriggers({ onUndo, onRedo, onPaste, onGroup, onUngroup, editable: false });
      // The context menu already withholds undo and redo from a viewer. The keyboard
      // route has to agree, or a read-only diagram rewrites itself on Control+Z.
      press("ControlLeft", "KeyZ");
      press("ControlLeft", "ShiftLeft", "KeyZ");
      press("ControlLeft", "KeyV");
      press("ControlLeft", "KeyG");
      press("ControlLeft", "KeyU");
      expect(onUndo).not.toHaveBeenCalled();
      expect(onRedo).not.toHaveBeenCalled();
      expect(onPaste).not.toHaveBeenCalled();
      expect(onGroup).not.toHaveBeenCalled();
      expect(onUngroup).not.toHaveBeenCalled();
    });

    it("should keep a read-only diagram navigable", () => {
      const onCopy = vi.fn();
      const onSelectAll = vi.fn();
      const onClearSelection = vi.fn();
      renderTriggers({ onCopy, onSelectAll, onClearSelection, editable: false });
      // Copying and selecting read the diagram rather than write it, so they survive
      // the read-only gate.
      press("ControlLeft", "KeyC");
      press("ControlLeft", "KeyA");
      press("Escape");
      expect(onCopy).toHaveBeenCalledOnce();
      expect(onSelectAll).toHaveBeenCalledOnce();
      expect(onClearSelection).toHaveBeenCalledOnce();
    });
  });
});
