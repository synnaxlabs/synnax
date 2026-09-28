// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useWindowResize } from "@/hooks/useResize";

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
