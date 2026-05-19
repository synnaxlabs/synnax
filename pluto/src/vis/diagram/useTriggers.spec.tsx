// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Triggers } from "@/triggers";
import { useTriggers, type UseTriggersProps } from "@/vis/diagram/useTriggers";

const TestComponent = (props: UseTriggersProps): ReactElement => {
  useTriggers(props);
  return <div>Canvas</div>;
};

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

describe("Diagram.useTriggers", () => {
  it("should call onUndo when Control+Z is pressed", () => {
    const onUndo = vi.fn();
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onUndo={onUndo} />
      </Triggers.Provider>,
    );
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyZ", ctrlKey: true });
    expect(onUndo).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyZ" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  it("should call onRedo when Control+Shift+Z is pressed", () => {
    const onRedo = vi.fn();
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onRedo={onRedo} />
      </Triggers.Provider>,
    );
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
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onCopy={onCopy} />
      </Triggers.Provider>,
    );
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyC", ctrlKey: true });
    expect(onCopy).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyC" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  it("should call onPaste when Control+V is pressed", () => {
    const onPaste = vi.fn();
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onPaste={onPaste} />
      </Triggers.Provider>,
    );
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyV", ctrlKey: true });
    expect(onPaste).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyV" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });

  it("should call onClear when Escape is pressed", () => {
    const onClear = vi.fn();
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onClearSelection={onClear} />
      </Triggers.Provider>,
    );
    fireEvent.keyDown(document.body, { code: "Escape" });
    expect(onClear).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "Escape" });
  });

  it("should call onSelectAll when Control+A is pressed", () => {
    const onSelectAll = vi.fn();
    render(
      <Triggers.Provider>
        <TestComponent {...defaultProps} onSelectAll={onSelectAll} />
      </Triggers.Provider>,
    );
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "KeyA", ctrlKey: true });
    expect(onSelectAll).toHaveBeenCalledOnce();
    fireEvent.keyUp(document.body, { code: "KeyA" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
  });
});
