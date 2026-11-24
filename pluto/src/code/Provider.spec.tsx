// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import {
  act,
  cleanup,
  render,
  renderHook,
  screen,
  waitFor,
} from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, Suspense } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { Code } from "@/code";

const { initialize, registerLinkOpener } = vi.hoisted(() => ({
  initialize: vi.fn(),
  registerLinkOpener: vi.fn(),
}));

// The real runtime does not load in jsdom, so each module the bootstrap imports is a
// stub, and initialize records the start.
vi.mock("@codingame/monaco-vscode-api", () => ({
  initialize,
  LogLevel: { Warning: 4 },
}));
vi.mock("@codingame/monaco-vscode-api/extensions", () => ({
  ExtensionHostKind: { LocalProcess: 1 },
  registerExtension: () => ({ registerFileUrl: () => {} }),
}));
vi.mock("@codingame/monaco-vscode-editor-api", () => ({
  editor: { registerLinkOpener },
}));
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
const loadProvider = async () => (await import("@/code")).Code;

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
    registerLinkOpener.mockClear();
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
      const language: Code.Language = {
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

    describe("openLink", () => {
      interface Link {
        scheme: string;
        toString: () => string;
      }

      // A link as Monaco hands it to an opener.
      const link = (url: string): Link => ({
        scheme: url.split(":")[0],
        toString: () => url,
      });

      /** Opens an editor under a provider that shows each status message. */
      const renderEditor = async (
        openLink?: (url: string) => Promise<void>,
      ): Promise<void> => {
        // Loaded after the reset, so the provider reads the same status context.
        const { Status } = await import("@synnaxlabs/lyra/status");
        const { Provider, useMonaco } = await loadProvider();
        const Editor = (): null => {
          useMonaco();
          return null;
        };
        const Notifications = (): ReactElement => (
          <>
            {Status.useNotifications().statuses.map(({ key, message }) => (
              <p key={key}>{message}</p>
            ))}
          </>
        );
        await act(async () =>
          render(
            <Status.Aggregator log={() => {}}>
              <Provider openLink={openLink}>
                <Notifications />
                <Suspense fallback={null}>
                  <Editor />
                </Suspense>
              </Provider>
            </Status.Aggregator>,
          ),
        );
        await waitFor(() => expect(initialize).toHaveBeenCalledOnce());
      };

      /** @returns the opener that the provider gives Monaco for openLink. */
      const renderOpener = async (
        openLink: (url: string) => Promise<void>,
      ): Promise<{ open: (resource: Link) => boolean }> => {
        await renderEditor(openLink);
        await waitFor(() => expect(registerLinkOpener).toHaveBeenCalledOnce());
        return registerLinkOpener.mock.calls[0][0];
      };

      it("should open a web link with openLink", async () => {
        const openLink = vi.fn(async () => {});
        const opener = await renderOpener(openLink);
        expect(opener.open(link("https://example.com/w2000#top"))).toBe(true);
        expect(openLink).toHaveBeenCalledExactlyOnceWith(
          "https://example.com/w2000#top",
        );
      });

      it("should leave a link that is not a web link to Monaco", async () => {
        const openLink = vi.fn(async () => {});
        const opener = await renderOpener(openLink);
        expect(opener.open(link("command:editor.action.selectAll"))).toBe(false);
        expect(openLink).not.toHaveBeenCalled();
      });

      it("should show a status when openLink fails", async () => {
        const openLink = vi.fn(async () => {
          throw new Error("no browser");
        });
        const opener = await renderOpener(openLink);
        act(() => void opener.open(link("https://example.com/w2000")));
        expect(await screen.findByText("Failed to open link")).toBeDefined();
      });

      it("should leave every link to Monaco when there is no openLink", async () => {
        await renderEditor();
        expect(registerLinkOpener).not.toHaveBeenCalled();
      });
    });
  });
});
