// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { initialize } = vi.hoisted(() => ({ initialize: vi.fn() }));

// The real runtime does not load in jsdom, so each module the bootstrap imports is a
// stub, and initialize records the start.
vi.mock("@codingame/monaco-vscode-api", () => ({ initialize }));
vi.mock("@codingame/monaco-vscode-editor-api", () => ({}));
vi.mock("@codingame/monaco-vscode-extension-api/localExtensionHost", () => ({}));
vi.mock("@codingame/monaco-vscode-languages-service-override", () => ({
  default: () => ({}),
}));
vi.mock("@codingame/monaco-vscode-textmate-service-override", () => ({
  default: () => ({}),
}));
vi.mock("@codingame/monaco-vscode-theme-defaults-default-extension", () => ({}));
vi.mock("@codingame/monaco-vscode-theme-service-override", () => ({
  default: () => ({}),
}));

// The runtime starts once per module instance, so each spec loads a fresh one.
const loadProvider = async () => await import("@/code/Provider");

describe("usePreload", () => {
  beforeEach(() => {
    vi.resetModules();
    initialize.mockClear();
    vi.useFakeTimers();
  });

  afterEach(() => {
    // Unmount while the stubbed idle callbacks still exist.
    cleanup();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("should start Monaco after the delay when the browser has no idle callbacks", async () => {
    const { Provider, usePreload } = await loadProvider();
    renderHook(() => usePreload(), { wrapper: Provider });
    await act(async () => await vi.advanceTimersByTimeAsync(1999));
    expect(initialize).not.toHaveBeenCalled();
    await act(async () => await vi.advanceTimersByTimeAsync(1));
    expect(initialize).toHaveBeenCalledOnce();
  });

  it("should start Monaco when the browser goes idle", async () => {
    let onIdle: (() => void) | undefined;
    const requestIdleCallback = vi.fn((callback: () => void) => {
      onIdle = callback;
      return 1;
    });
    vi.stubGlobal("requestIdleCallback", requestIdleCallback);
    vi.stubGlobal("cancelIdleCallback", vi.fn());
    const { Provider, usePreload } = await loadProvider();
    renderHook(() => usePreload(), { wrapper: Provider });
    expect(requestIdleCallback).toHaveBeenCalledWith(expect.any(Function), {
      timeout: 2000,
    });
    expect(initialize).not.toHaveBeenCalled();
    await act(async () => {
      onIdle?.();
      await vi.runAllTimersAsync();
    });
    expect(initialize).toHaveBeenCalledOnce();
  });

  it("should not start Monaco when it unmounts before the page is idle", async () => {
    const { Provider, usePreload } = await loadProvider();
    const { unmount } = renderHook(() => usePreload(), { wrapper: Provider });
    unmount();
    await act(async () => await vi.advanceTimersByTimeAsync(2000));
    expect(initialize).not.toHaveBeenCalled();
  });
});
