// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, renderHook } from "@testing-library/react";
import { type MouseEvent } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Button } from "@/button";

const press = (button = 0): MouseEvent<HTMLButtonElement> =>
  ({ button }) as MouseEvent<HTMLButtonElement>;

const release = (): void => {
  fireEvent.mouseUp(document);
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
      const { result } = renderHook(() => Button.useHold({ onClick }));
      const e = press();
      act(() => result.current.onClick(e));
      expect(onClick).toHaveBeenCalledWith(e);
      expect(result.current.delay.isZero).toBe(true);
    });

    it("should still track a primary press", () => {
      const { result } = renderHook(() => Button.useHold({}));
      act(() => result.current.onMouseDown(press()));
      expect(result.current.pressed).toBe(true);
      act(release);
      expect(result.current.pressed).toBe(false);
    });
  });

  describe("with a delay", () => {
    it("should swallow a plain click", () => {
      const onClick = vi.fn();
      const { result } = renderHook(() =>
        Button.useHold({ onClick, onClickDelay: 500 }),
      );
      act(() => result.current.onClick(press()));
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should fire once the press is held for the delay", () => {
      const onClick = vi.fn();
      const { result } = renderHook(() =>
        Button.useHold({ onClick, onClickDelay: 500 }),
      );
      const e = press();
      act(() => result.current.onMouseDown(e));
      act(() => {
        vi.advanceTimersByTime(499);
      });
      expect(onClick).not.toHaveBeenCalled();
      act(() => {
        vi.advanceTimersByTime(1);
      });
      expect(onClick).toHaveBeenCalledExactlyOnceWith(e);
    });

    it("should cancel on a release before the delay", () => {
      const onClick = vi.fn();
      const { result } = renderHook(() =>
        Button.useHold({ onClick, onClickDelay: 500 }),
      );
      act(() => result.current.onMouseDown(press()));
      act(release);
      act(() => {
        vi.advanceTimersByTime(1000);
      });
      expect(onClick).not.toHaveBeenCalled();
      expect(result.current.pressed).toBe(false);
    });

    it("should stay pressed after firing until the release", () => {
      const { result } = renderHook(() => Button.useHold({ onClickDelay: 500 }));
      act(() => result.current.onMouseDown(press()));
      act(() => {
        vi.advanceTimersByTime(500);
      });
      expect(result.current.pressed).toBe(true);
      act(release);
      expect(result.current.pressed).toBe(false);
    });

    it("should restart the hold on a second press", () => {
      const onClick = vi.fn();
      const { result } = renderHook(() =>
        Button.useHold({ onClick, onClickDelay: 500 }),
      );
      act(() => result.current.onMouseDown(press()));
      act(() => {
        vi.advanceTimersByTime(400);
      });
      act(() => result.current.onMouseDown(press()));
      act(() => {
        vi.advanceTimersByTime(400);
      });
      expect(onClick).not.toHaveBeenCalled();
      act(() => {
        vi.advanceTimersByTime(100);
      });
      expect(onClick).toHaveBeenCalledOnce();
    });
  });

  it("should forward onMouseDown for every button", () => {
    const onMouseDown = vi.fn();
    const { result } = renderHook(() => Button.useHold({ onMouseDown }));
    const e = press(2);
    act(() => result.current.onMouseDown(e));
    expect(onMouseDown).toHaveBeenCalledWith(e);
  });

  it("should ignore a secondary-button press", () => {
    const onClick = vi.fn();
    const { result } = renderHook(() => Button.useHold({ onClick, onClickDelay: 500 }));
    act(() => result.current.onMouseDown(press(2)));
    expect(result.current.pressed).toBe(false);
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(onClick).not.toHaveBeenCalled();
  });

  describe("cleanup", () => {
    it("should cancel the hold on unmount", () => {
      const onClick = vi.fn();
      const { result, unmount } = renderHook(() =>
        Button.useHold({ onClick, onClickDelay: 500 }),
      );
      act(() => result.current.onMouseDown(press()));
      unmount();
      act(() => {
        vi.advanceTimersByTime(1000);
      });
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should cancel the hold when disabled mid-press", () => {
      const onClick = vi.fn();
      const { result, rerender } = renderHook(
        ({ disabled }) => Button.useHold({ onClick, onClickDelay: 500, disabled }),
        { initialProps: { disabled: false } },
      );
      act(() => result.current.onMouseDown(press()));
      rerender({ disabled: true });
      expect(result.current.pressed).toBe(false);
      act(() => {
        vi.advanceTimersByTime(1000);
      });
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should ignore a press while disabled", () => {
      const onClick = vi.fn();
      const { result } = renderHook(() =>
        Button.useHold({ onClick, onClickDelay: 500, disabled: true }),
      );
      act(() => result.current.onMouseDown(press()));
      expect(result.current.pressed).toBe(false);
      act(() => {
        vi.advanceTimersByTime(1000);
      });
      expect(onClick).not.toHaveBeenCalled();
    });
  });
});
