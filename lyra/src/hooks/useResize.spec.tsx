// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render, renderHook } from "@testing-library/react";
import type { ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useResize, type UseResizeHandler, useWindowResize } from "@/hooks";

const Observed = ({ onResize }: { onResize: UseResizeHandler }): ReactElement => (
  <div ref={useResize(onResize)} />
);

describe("useResize", () => {
  let Base: typeof ResizeObserver;
  let created = 0;

  beforeEach(() => {
    Base = globalThis.ResizeObserver;
    created = 0;
    vi.stubGlobal(
      "ResizeObserver",
      class extends Base {
        constructor(callback: ResizeObserverCallback) {
          super(callback);
          created++;
        }
      },
    );
  });

  afterEach(() => {
    vi.stubGlobal("ResizeObserver", Base);
  });

  it("should create one observer on mount", () => {
    render(<Observed onResize={vi.fn()} />);
    expect(created).toBe(1);
  });

  it("should create no observers across rerenders with stable inputs", () => {
    const onResize = vi.fn();
    const { rerender } = render(<Observed onResize={onResize} />);
    created = 0;
    rerender(<Observed onResize={onResize} />);
    rerender(<Observed onResize={onResize} />);
    expect(created).toBe(0);
  });

  it("should create one observer when the handler changes", () => {
    const { rerender } = render(<Observed onResize={vi.fn()} />);
    created = 0;
    rerender(<Observed onResize={vi.fn()} />);
    expect(created).toBe(1);
  });
});

describe("useWindowResize", () => {
  it("should call the handler on window resize when enabled", () => {
    const handler = vi.fn();
    renderHook(() => useWindowResize(handler));
    act(() => {
      window.dispatchEvent(new Event("resize"));
    });
    expect(handler).toHaveBeenCalled();
  });

  it("should not call the handler when enabled is false", () => {
    const handler = vi.fn();
    renderHook(() => useWindowResize(handler, { enabled: false }));
    act(() => {
      window.dispatchEvent(new Event("resize"));
    });
    expect(handler).not.toHaveBeenCalled();
  });

  it("should attach the listener when enabled transitions to true", () => {
    const handler = vi.fn();
    const { rerender } = renderHook(
      ({ enabled }) => useWindowResize(handler, { enabled }),
      { initialProps: { enabled: false } },
    );
    act(() => {
      window.dispatchEvent(new Event("resize"));
    });
    expect(handler).not.toHaveBeenCalled();
    rerender({ enabled: true });
    act(() => {
      window.dispatchEvent(new Event("resize"));
    });
    expect(handler).toHaveBeenCalled();
  });

  it("should detach the listener when enabled transitions to false", () => {
    const handler = vi.fn();
    const { rerender } = renderHook(
      ({ enabled }) => useWindowResize(handler, { enabled }),
      { initialProps: { enabled: true } },
    );
    act(() => {
      window.dispatchEvent(new Event("resize"));
    });
    expect(handler).toHaveBeenCalledTimes(1);
    rerender({ enabled: false });
    act(() => {
      window.dispatchEvent(new Event("resize"));
    });
    expect(handler).toHaveBeenCalledTimes(1);
  });
});
