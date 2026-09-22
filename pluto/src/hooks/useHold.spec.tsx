// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  renderHook,
  type RenderResult,
} from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useHold, type UseHoldProps } from "@/hooks/useHold";

const Host = (props: UseHoldProps<HTMLButtonElement>): ReactElement => {
  const hold = useHold(props);
  return (
    <button
      aria-pressed={hold.pressed}
      onClick={hold.onClick}
      onMouseDown={hold.onMouseDown}
      onKeyDown={hold.onKeyDown}
      onKeyUp={hold.onKeyUp}
    >
      <span>{hold.delay.milliseconds}</span>
    </button>
  );
};

const renderHold = (
  props: UseHoldProps<HTMLButtonElement> = {},
): RenderResult & { button: HTMLElement } => {
  const result = render(<Host {...props} />);
  return { ...result, button: result.getByRole("button") };
};

const pressed = (button: HTMLElement): boolean =>
  button.getAttribute("aria-pressed") === "true";

// The hold's own release is a state update outside any event, so act flushes it.
const advance = (ms: number): void => {
  act(() => {
    vi.advanceTimersByTime(ms);
  });
};

describe("useHold", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("without a delay", () => {
    it("should pass a click straight through", () => {
      const onClick = vi.fn();
      const { button } = renderHold({ onClick });
      fireEvent.click(button);
      expect(onClick).toHaveBeenCalledOnce();
      expect(button.textContent).toBe("0");
    });

    it("should still track a primary press", () => {
      const { button } = renderHold();
      fireEvent.mouseDown(button);
      expect(pressed(button)).toBe(true);
      fireEvent.mouseUp(document);
      expect(pressed(button)).toBe(false);
    });
  });

  describe("delay", () => {
    it("should keep the same TimeSpan across equal crude values", () => {
      const initialProps: UseHoldProps<HTMLElement> = {
        onClickDelay: TimeSpan.milliseconds(500),
      };
      const { result, rerender } = renderHook(useHold, { initialProps });
      const first = result.current.delay;
      rerender({ onClickDelay: TimeSpan.milliseconds(500) });
      expect(result.current.delay).toBe(first);
      rerender({ onClickDelay: 500 });
      expect(result.current.delay).toBe(first);
      rerender({ onClickDelay: 501 });
      expect(result.current.delay).not.toBe(first);
      expect(result.current.delay.milliseconds).toBe(501);
    });
  });

  describe("with a delay", () => {
    it("should swallow a plain click", () => {
      const onClick = vi.fn();
      const { button } = renderHold({ onClick, onClickDelay: 500 });
      fireEvent.click(button);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should fire with the press once it is held for the delay", () => {
      const onClick = vi.fn();
      const { button } = renderHold({ onClick, onClickDelay: 500 });
      fireEvent.mouseDown(button);
      advance(499);
      expect(onClick).not.toHaveBeenCalled();
      advance(1);
      expect(onClick).toHaveBeenCalledExactlyOnceWith(
        expect.objectContaining({ type: "mousedown" }),
      );
    });

    it("should cancel on a release before the delay", () => {
      const onClick = vi.fn();
      const { button } = renderHold({ onClick, onClickDelay: 500 });
      fireEvent.mouseDown(button);
      fireEvent.mouseUp(document);
      advance(1000);
      expect(onClick).not.toHaveBeenCalled();
      expect(pressed(button)).toBe(false);
    });

    it("should release itself when the hold fires", () => {
      const onClick = vi.fn();
      const { button } = renderHold({ onClick, onClickDelay: 500 });
      fireEvent.mouseDown(button);
      advance(500);
      expect(onClick).toHaveBeenCalledOnce();
      expect(pressed(button)).toBe(false);
      fireEvent.mouseUp(document);
      advance(1000);
      expect(onClick).toHaveBeenCalledOnce();
    });

    it("should fire the onClick current when the delay elapses", () => {
      const stale = vi.fn();
      const fresh = vi.fn();
      const { button, rerender } = renderHold({ onClick: stale, onClickDelay: 500 });
      fireEvent.mouseDown(button);
      rerender(<Host onClick={fresh} onClickDelay={500} />);
      advance(500);
      expect(stale).not.toHaveBeenCalled();
      expect(fresh).toHaveBeenCalledOnce();
    });

    it("should restart the hold on a second press", () => {
      const onClick = vi.fn();
      const { button } = renderHold({ onClick, onClickDelay: 500 });
      fireEvent.mouseDown(button);
      advance(400);
      fireEvent.mouseDown(button);
      advance(400);
      expect(onClick).not.toHaveBeenCalled();
      advance(100);
      expect(onClick).toHaveBeenCalledOnce();
    });
  });

  it("should forward onMouseDown for every button", () => {
    const onMouseDown = vi.fn();
    const { button } = renderHold({ onMouseDown });
    fireEvent.mouseDown(button, { button: 2 });
    expect(onMouseDown).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ button: 2 }),
    );
  });

  it("should ignore a secondary-button press", () => {
    const onClick = vi.fn();
    const { button } = renderHold({ onClick, onClickDelay: 500 });
    fireEvent.mouseDown(button, { button: 2 });
    expect(pressed(button)).toBe(false);
    advance(1000);
    expect(onClick).not.toHaveBeenCalled();
  });

  describe("cleanup", () => {
    it("should cancel the hold on unmount", () => {
      const onClick = vi.fn();
      const { button, unmount } = renderHold({ onClick, onClickDelay: 500 });
      fireEvent.mouseDown(button);
      unmount();
      advance(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should cancel the hold when disabled mid-press", () => {
      const onClick = vi.fn();
      const { button, rerender } = renderHold({ onClick, onClickDelay: 500 });
      fireEvent.mouseDown(button);
      rerender(<Host onClick={onClick} onClickDelay={500} disabled />);
      expect(pressed(button)).toBe(false);
      advance(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should ignore a press while disabled", () => {
      const onClick = vi.fn();
      const { button } = renderHold({ onClick, onClickDelay: 500, disabled: true });
      fireEvent.mouseDown(button);
      expect(pressed(button)).toBe(false);
      advance(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should release the hold when the window blurs", () => {
      const onClick = vi.fn();
      const { button } = renderHold({ onClick, onClickDelay: 500 });
      fireEvent.mouseDown(button);
      fireEvent.blur(window);
      expect(pressed(button)).toBe(false);
      advance(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should release the hold when a drag starts", () => {
      const onClick = vi.fn();
      const { button } = renderHold({ onClick, onClickDelay: 500 });
      fireEvent.mouseDown(button);
      fireEvent.dragStart(document);
      expect(pressed(button)).toBe(false);
      advance(1000);
      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe("keyboard", () => {
    it.each([" ", "Enter"])("should mark %j and clear it on keyup", (key) => {
      const { button } = renderHold();
      fireEvent.keyDown(button, { key });
      expect(pressed(button)).toBe(true);
      fireEvent.keyUp(button, { key });
      expect(pressed(button)).toBe(false);
    });

    it("should ignore other keys", () => {
      const { button } = renderHold();
      fireEvent.keyDown(button, { key: "a" });
      expect(pressed(button)).toBe(false);
    });

    it("should ignore a prevented keydown", () => {
      const { button } = renderHold();
      const e = new KeyboardEvent("keydown", {
        key: " ",
        bubbles: true,
        cancelable: true,
      });
      e.preventDefault();
      fireEvent(button, e);
      expect(pressed(button)).toBe(false);
    });

    it("should ignore a keydown from a nested target", () => {
      const { button } = renderHold();
      fireEvent.keyDown(button.firstElementChild ?? button, { key: " " });
      expect(pressed(button)).toBe(false);
    });

    it("should not mark a delayed control", () => {
      const onClick = vi.fn();
      const { button } = renderHold({ onClick, onClickDelay: 500 });
      fireEvent.keyDown(button, { key: " " });
      expect(pressed(button)).toBe(false);
      advance(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should not mark a disabled control", () => {
      const { button } = renderHold({ disabled: true });
      fireEvent.keyDown(button, { key: " " });
      expect(pressed(button)).toBe(false);
    });
  });
});
