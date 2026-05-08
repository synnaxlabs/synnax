// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, useRef } from "react";
import { describe, expect, it, vi } from "vitest";

import { mockBoundingClientRect } from "@/testutil/dom";
import { Triggers } from "@/triggers";
import { useTriggers, type UseTriggersProps } from "@/vis/diagram/useTriggers";

interface TestComponentProps extends Omit<UseTriggersProps, "region"> {}

const TestComponent = (props: TestComponentProps): ReactElement => {
  const regionRef = useRef<HTMLDivElement>(null);
  useTriggers({ ...props, region: regionRef });
  return <div ref={regionRef}>Canvas</div>;
};

const noop = () => {};
const noopCursor = () => {};

const defaultProps: TestComponentProps = {
  onUndo: noop,
  onRedo: noop,
  onCopy: noopCursor,
  onPaste: noopCursor,
  onClear: noop,
  onSelectAll: noop,
};

describe("Diagram.useTriggers", () => {
  it("should call onUndo when Control+Z is pressed", () => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    const onUndo = vi.fn();
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onUndo={onUndo} />
      </Triggers.Provider>,
    );
    fireEvent.mouseMove(document.body, { clientX: 10, clientY: 10 });
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyZ", ctrlKey: true });
    expect(onUndo).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyZ" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  it("should call onRedo when Control+Shift+Z is pressed", () => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    const onRedo = vi.fn();
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onRedo={onRedo} />
      </Triggers.Provider>,
    );
    fireEvent.mouseMove(document.body, { clientX: 10, clientY: 10 });
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
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    const onCopy = vi.fn();
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onCopy={onCopy} />
      </Triggers.Provider>,
    );
    fireEvent.mouseMove(document.body, { clientX: 10, clientY: 10 });
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyC", ctrlKey: true });
    expect(onCopy).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyC" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  it("should call onPaste when Control+V is pressed", () => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    const onPaste = vi.fn();
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onPaste={onPaste} />
      </Triggers.Provider>,
    );
    fireEvent.mouseMove(document.body, { clientX: 10, clientY: 10 });
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyV", ctrlKey: true });
    expect(onPaste).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyV" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  it("should call onClear when Escape is pressed", () => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    const onClear = vi.fn();
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onClear={onClear} />
      </Triggers.Provider>,
    );
    fireEvent.mouseMove(document.body, { clientX: 10, clientY: 10 });
    fireEvent.keyDown(document.body, { code: "Escape" });
    expect(onClear).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "Escape" });
  });

  it("should call onSelectAll when Control+A is pressed", () => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    const onSelectAll = vi.fn();
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onSelectAll={onSelectAll} />
      </Triggers.Provider>,
    );
    fireEvent.mouseMove(document.body, { clientX: 10, clientY: 10 });
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyA", ctrlKey: true });
    expect(onSelectAll).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyA" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  describe("disabled", () => {
    it("should not call any callbacks when disabled is true", () => {
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
      const onUndo = vi.fn();
      const onCopy = vi.fn();
      const onPaste = vi.fn();
      const onClear = vi.fn();
      const onSelectAll = vi.fn();
      render(
        <Triggers.Provider>
          <TestComponent
            {...defaultProps}
            onUndo={onUndo}
            onCopy={onCopy}
            onPaste={onPaste}
            onClear={onClear}
            onSelectAll={onSelectAll}
            disabled
          />
        </Triggers.Provider>,
      );
      fireEvent.mouseMove(document.body, { clientX: 10, clientY: 10 });

      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyZ", ctrlKey: true });
      fireEvent.keyUp(document.body, { code: "KeyZ" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });

      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyC", ctrlKey: true });
      fireEvent.keyUp(document.body, { code: "KeyC" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });

      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyV", ctrlKey: true });
      fireEvent.keyUp(document.body, { code: "KeyV" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });

      fireEvent.keyDown(document.body, { code: "Escape" });
      fireEvent.keyUp(document.body, { code: "Escape" });

      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyA", ctrlKey: true });
      fireEvent.keyUp(document.body, { code: "KeyA" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });

      expect(onUndo).not.toHaveBeenCalled();
      expect(onCopy).not.toHaveBeenCalled();
      expect(onPaste).not.toHaveBeenCalled();
      expect(onClear).not.toHaveBeenCalled();
      expect(onSelectAll).not.toHaveBeenCalled();
    });

    it("should not fire when disabled is initially false then becomes true", () => {
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
      const onUndo = vi.fn();
      const { rerender } = render(
        <Triggers.Provider>
          <TestComponent {...defaultProps} onUndo={onUndo} />
        </Triggers.Provider>,
      );
      fireEvent.mouseMove(document.body, { clientX: 10, clientY: 10 });

      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyZ", ctrlKey: true });
      expect(onUndo).toHaveBeenCalledOnce();
      fireEvent.keyUp(document.body, { code: "KeyZ" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });

      rerender(
        <Triggers.Provider>
          <TestComponent {...defaultProps} onUndo={onUndo} disabled />
        </Triggers.Provider>,
      );

      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyZ", ctrlKey: true });
      expect(onUndo).toHaveBeenCalledOnce();
      fireEvent.keyUp(document.body, { code: "KeyZ" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });
    });

    it("should resume firing when disabled changes from true to false", () => {
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
      const onUndo = vi.fn();
      const { rerender } = render(
        <Triggers.Provider>
          <TestComponent {...defaultProps} onUndo={onUndo} disabled />
        </Triggers.Provider>,
      );
      fireEvent.mouseMove(document.body, { clientX: 10, clientY: 10 });

      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyZ", ctrlKey: true });
      expect(onUndo).not.toHaveBeenCalled();
      fireEvent.keyUp(document.body, { code: "KeyZ" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });

      rerender(
        <Triggers.Provider>
          <TestComponent {...defaultProps} onUndo={onUndo} disabled={false} />
        </Triggers.Provider>,
      );

      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyZ", ctrlKey: true });
      expect(onUndo).toHaveBeenCalledOnce();
      fireEvent.keyUp(document.body, { code: "KeyZ" });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });
    });
  });
});
