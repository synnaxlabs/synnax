// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, cleanup, render, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, Suspense } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { type Language } from "@/code/language";

const { initialize } = vi.hoisted(() => ({ initialize: vi.fn() }));

// The real runtime does not load in jsdom, so each module the bootstrap imports is a
// stub, and initialize records the start.
vi.mock("@codingame/monaco-vscode-api", () => ({ initialize }));
vi.mock("@codingame/monaco-vscode-api/extensions", () => ({
  ExtensionHostKind: { LocalProcess: 1 },
  registerExtension: () => ({ registerFileUrl: () => {} }),
}));
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

// Returns a function that makes the page go idle.
const stubIdleCallbacks = (): (() => void) => {
  let onIdle: (() => void) | undefined;
  vi.stubGlobal(
    "requestIdleCallback",
    vi.fn((callback: () => void) => {
      onIdle = callback;
      return 1;
    }),
  );
  vi.stubGlobal("cancelIdleCallback", vi.fn());
  return () => onIdle?.();
};

describe("Code", () => {
  beforeEach(() => {
    vi.resetModules();
    initialize.mockClear();
  });

  afterEach(() => {
    // Unmount while the stubbed idle callbacks still exist.
    cleanup();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  describe("usePreload", () => {
    beforeEach(() => {
      vi.useFakeTimers();
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
      const goIdle = stubIdleCallbacks();
      const { Provider, usePreload } = await loadProvider();
      renderHook(() => usePreload(), { wrapper: Provider });
      expect(requestIdleCallback).toHaveBeenCalledWith(expect.any(Function), {
        timeout: 2000,
      });
      expect(initialize).not.toHaveBeenCalled();
      await act(async () => {
        goIdle();
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

  describe("Provider", () => {
    it("should connect a language server only once an editor opens", async () => {
      const goIdle = stubIdleCallbacks();
      const languageServer = vi.fn(() => new Promise<never>(() => {}));
      const language: Language = {
        name: "test",
        configuration: "{}",
        grammar: { scopeName: "source.test", raw: "{}" },
        languageServer,
      };
      // Loaded after the reset, so the provider reads the same Synnax context.
      const { createSynnaxWrapper } = await import("@/testutil");
      const { Provider, useMonaco, usePreload } = await loadProvider();
      const SynnaxWrapper = createSynnaxWrapper({ client: createTestClient() });
      const Preload = (): null => {
        usePreload();
        return null;
      };
      const Editor = (): null => {
        useMonaco();
        return null;
      };
      const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <SynnaxWrapper>
          <Provider languages={[language]}>
            <Preload />
            <Suspense fallback={null}>{children}</Suspense>
          </Provider>
        </SynnaxWrapper>
      );
      const { rerender } = render(<Wrapper />);
      act(() => goIdle());
      await waitFor(() => expect(initialize).toHaveBeenCalledOnce());
      expect(languageServer).not.toHaveBeenCalled();
      await act(async () =>
        rerender(
          <Wrapper>
            <Editor />
          </Wrapper>,
        ),
      );
      await waitFor(() => expect(languageServer).toHaveBeenCalledOnce());
    });
  });
});
