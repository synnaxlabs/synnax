// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { scheduler } from "@synnaxlabs/x";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => {
  let optedOut = false;
  let label = "some-child-window";
  let session = "";
  return {
    engine: "tauri" as const,
    setOptedOut: (next: boolean) => (optedOut = next),
    label: () => label,
    setLabel: (next: string) => (label = next),
    /** Moves PostHog to a session on its own, as after an idle time. */
    moveSession: (next: string) => (session = next),
    posthog: {
      init: vi.fn((_key: string, config: { bootstrap: { sessionID: string } }) => {
        session = config.bootstrap.sessionID;
      }),
      set_config: vi.fn(),
      register: vi.fn(),
      capture: vi.fn(),
      identify: vi.fn(),
      reset: vi.fn(),
      has_opted_out_capturing: vi.fn(() => optedOut),
      opt_in_capturing: vi.fn(() => (optedOut = false)),
      opt_out_capturing: vi.fn(() => (optedOut = true)),
      onSessionId: vi.fn<(share: (id: string) => void) => void>(),
      get_session_id: vi.fn(() => session),
      sessionManager: {
        setBootstrapSessionId: vi.fn((next: string) => {
          session = next;
          return true;
        }),
      },
    },
  };
});

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

// A window other than main unless a spec says so, so only the account and the usage
// data choice report.
vi.mock("@tauri-apps/api/window", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  getCurrentWindow: () => ({ label: mocks.label() }),
}));

// PostHog sends from init on, so it is the network boundary of this spec.
vi.mock("posthog-js/dist/module.no-external", () => ({ default: mocks.posthog }));
vi.mock("posthog-js/dist/surveys", () => ({}));
vi.mock("posthog-js/dist/exception-autocapture", () => ({}));
vi.mock("posthog-js/dist/posthog-recorder", () => ({}));

import { Analytics } from "@/app/analytics";
import { Analytics as PlatformAnalytics } from "@/platform/analytics";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

const INSTALL = {
  id: "install_a",
  firstLaunch: false,
  hoursSinceLastLaunch: 1,
  erasedBytes: null,
  os: "macos",
  arch: "aarch64",
};

const UUID_V7 = /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

const LINK = { secret: "shh", email: "a@example.com", user: "user_a" };

/** The session the main window started, as a UUIDv7. */
const SESSION = "019a2b3c-4d5e-7f60-8a1b-2c3d4e5f6a7b";

/** A later session, after an idle time or the length limit. */
const NEXT = "019a2b3c-4d5e-7f60-8a1b-000000000000";

const renderProvider = async (
  account: Partial<Session.Account.SliceState>,
  analytics: Session.Analytics.SliceState = { sessionID: SESSION },
) => {
  const { wrapper: Base, store } = await createConsoleWrapper({
    client: null,
    preloadedState: {
      [Session.Account.SLICE_NAME]: { ...Session.Account.ZERO_SLICE_STATE, ...account },
      [Session.Analytics.SLICE_NAME]: analytics,
    },
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Base>
      <Analytics.Provider>{children}</Analytics.Provider>
    </Base>
  );
  const { result } = renderHook(() => PlatformAnalytics.use(), { wrapper: Wrapper });
  return { store, sink: () => result.current };
};

describe("Analytics.Provider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.setOptedOut(false);
    mocks.setLabel("some-child-window");
    mockIPC((cmd) => (cmd === "install_info" ? INSTALL : undefined));
  });

  afterEach(() => clearMocks());

  it("should load posthog only once the user allows usage data", async () => {
    const { store, sink } = await renderProvider({ link: LINK });
    sink().capture("plot_created", {});
    // Async effects start after a task queue flush, so wait one out before asserting.
    await act(async () => await scheduler.flushTaskQueue());
    expect(mocks.posthog.init).not.toHaveBeenCalled();
    act(() => {
      store.dispatch(Session.Account.setAnalyticsAllowed(true));
    });
    await waitFor(() => expect(mocks.posthog.init).toHaveBeenCalledOnce());
    // An event from before the choice stays dropped.
    expect(mocks.posthog.capture).not.toHaveBeenCalled();
  });

  it("should not start posthog when the user refuses while it loads", async () => {
    let answer: (install: typeof INSTALL) => void = () => {};
    mockIPC((cmd) =>
      cmd === "install_info"
        ? new Promise<typeof INSTALL>((resolve) => {
            answer = resolve;
          })
        : undefined,
    );
    const { store } = await renderProvider({ link: LINK, analyticsAllowed: true });
    act(() => {
      store.dispatch(Session.Account.setAnalyticsAllowed(false));
    });
    answer(INSTALL);
    await act(async () => await scheduler.flushTaskQueue());
    expect(mocks.posthog.init).not.toHaveBeenCalled();
  });

  it("should identify the account once posthog starts", async () => {
    await renderProvider({ link: LINK, analyticsAllowed: true });
    await waitFor(() =>
      expect(mocks.posthog.identify).toHaveBeenCalledWith("user_a", {
        email: "a@example.com",
      }),
    );
    expect(mocks.posthog.init.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.posthog.identify.mock.invocationCallOrder[0],
    );
  });

  it("should reset and opt out when the user refuses usage data", async () => {
    const { store, sink } = await renderProvider({
      link: LINK,
      analyticsAllowed: true,
    });
    await waitFor(() => expect(mocks.posthog.identify).toHaveBeenCalled());
    act(() => {
      store.dispatch(Session.Account.setAnalyticsAllowed(false));
    });
    expect(mocks.posthog.reset).toHaveBeenCalledOnce();
    expect(mocks.posthog.opt_out_capturing).toHaveBeenCalledOnce();
    // Flags go off before the reset reloads them, and the reset clears the stored
    // opt-out, so the opt-out comes last.
    expect(mocks.posthog.set_config).toHaveBeenCalledWith({
      advanced_disable_flags: true,
    });
    expect(mocks.posthog.set_config.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.posthog.reset.mock.invocationCallOrder[0],
    );
    expect(mocks.posthog.reset.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.posthog.opt_out_capturing.mock.invocationCallOrder[0],
    );
    sink().capture("plot_created", {});
    expect(mocks.posthog.capture).not.toHaveBeenCalled();
  });

  it("should stop posthog when the machine logs out", async () => {
    const { store } = await renderProvider({ link: LINK, analyticsAllowed: true });
    await waitFor(() => expect(mocks.posthog.identify).toHaveBeenCalled());
    act(() => {
      store.dispatch(Session.Account.clear());
    });
    expect(mocks.posthog.opt_out_capturing).toHaveBeenCalledOnce();
  });

  it("should send and identify again once usage data is allowed again", async () => {
    const { store, sink } = await renderProvider({
      link: LINK,
      analyticsAllowed: true,
    });
    await waitFor(() => expect(mocks.posthog.identify).toHaveBeenCalledOnce());
    act(() => {
      store.dispatch(Session.Account.setAnalyticsAllowed(false));
    });
    act(() => {
      store.dispatch(Session.Account.setAnalyticsAllowed(true));
    });
    expect(mocks.posthog.set_config).toHaveBeenLastCalledWith({
      advanced_disable_flags: false,
    });
    expect(mocks.posthog.opt_in_capturing).toHaveBeenCalledWith({
      captureEventName: false,
    });
    expect(mocks.posthog.identify).toHaveBeenCalledTimes(2);
    sink().capture("plot_created", {});
    expect(mocks.posthog.capture).toHaveBeenCalledWith("plot_created", {});
    expect(mocks.posthog.init).toHaveBeenCalledOnce();
  });

  it("should opt back in when an earlier launch opted out", async () => {
    mocks.setOptedOut(true);
    await renderProvider({ analyticsAllowed: true });
    await waitFor(() => expect(mocks.posthog.init).toHaveBeenCalled());
    expect(mocks.posthog.opt_in_capturing).toHaveBeenCalledWith({
      captureEventName: false,
    });
  });

  it("should reset before identifying a different account", async () => {
    // Without the reset, the second account's first events would carry the first
    // account's identity.
    const { store } = await renderProvider({ link: LINK, analyticsAllowed: true });
    await waitFor(() => expect(mocks.posthog.identify).toHaveBeenCalledOnce());
    act(() => {
      store.dispatch(
        Session.Account.link({
          secret: "shh",
          email: "b@example.com",
          user: "user_b",
          analyticsAllowed: true,
        }),
      );
    });
    expect(mocks.posthog.reset).toHaveBeenCalledExactlyOnceWith({
      bootstrap: { distinctID: "install_a", sessionID: SESSION },
    });
    expect(mocks.posthog.identify).toHaveBeenLastCalledWith("user_b", {
      email: "b@example.com",
    });
    expect(mocks.posthog.reset.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.posthog.identify.mock.invocationCallOrder[1],
    );
  });

  describe("session", () => {
    it("should join the session that the main window started", async () => {
      await renderProvider({ link: LINK, analyticsAllowed: true });
      await waitFor(() => expect(mocks.posthog.init).toHaveBeenCalledOnce());
      expect(mocks.posthog.init).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          bootstrap: { distinctID: "install_a", sessionID: SESSION },
        }),
      );
    });

    it("should wait for the main window to start a session", async () => {
      const { store } = await renderProvider(
        { link: LINK, analyticsAllowed: true },
        {},
      );
      await act(async () => await scheduler.flushTaskQueue());
      expect(mocks.posthog.init).not.toHaveBeenCalled();
      act(() => {
        store.dispatch(Session.Analytics.setSessionID(SESSION));
      });
      await waitFor(() => expect(mocks.posthog.init).toHaveBeenCalledOnce());
      expect(mocks.posthog.init).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          bootstrap: { distinctID: "install_a", sessionID: SESSION },
        }),
      );
    });

    it("should start a new session in the main window", async () => {
      mocks.setLabel(Drift.MAIN_WINDOW);
      mockIPC((cmd) => {
        if (cmd === "install_info") return INSTALL;
        if (cmd === "plugin:app|version") return "1.0.0";
        if (cmd === "supervisor_history")
          return {
            starts: 0,
            exits: 0,
            failures: 0,
            readies: 0,
            readyAt: null,
            timeToReadyMs: null,
            lastExit: null,
          };
        return undefined;
      });
      const { store } = await renderProvider({ analyticsAllowed: true }, {});
      await waitFor(() => expect(mocks.posthog.init).toHaveBeenCalledOnce());
      const sessionID = Session.Analytics.selectSessionID(store.getState());
      expect(sessionID).toMatch(UUID_V7);
      expect(mocks.posthog.init).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ bootstrap: { distinctID: "install_a", sessionID } }),
      );
    });

    it("should share a session that PostHog moves to", async () => {
      const { store } = await renderProvider({ link: LINK, analyticsAllowed: true });
      await waitFor(() => expect(mocks.posthog.onSessionId).toHaveBeenCalledOnce());
      const share = mocks.posthog.onSessionId.mock.calls[0][0];
      act(() => {
        mocks.moveSession(NEXT);
        share(NEXT);
      });
      expect(Session.Analytics.selectSessionID(store.getState())).toBe(NEXT);
      expect(mocks.posthog.sessionManager.setBootstrapSessionId).not.toHaveBeenCalled();
    });

    it("should move to a session that another window shares", async () => {
      const { store } = await renderProvider({ link: LINK, analyticsAllowed: true });
      await waitFor(() => expect(mocks.posthog.onSessionId).toHaveBeenCalledOnce());
      act(() => {
        store.dispatch(Session.Analytics.setSessionID(NEXT));
      });
      expect(
        mocks.posthog.sessionManager.setBootstrapSessionId,
      ).toHaveBeenCalledExactlyOnceWith(NEXT);
    });

    it("should keep the shared session when the user refuses usage data", async () => {
      const { store } = await renderProvider({ link: LINK, analyticsAllowed: true });
      await waitFor(() => expect(mocks.posthog.identify).toHaveBeenCalled());
      act(() => {
        store.dispatch(Session.Account.setAnalyticsAllowed(false));
      });
      expect(mocks.posthog.reset).toHaveBeenCalledExactlyOnceWith({
        bootstrap: { distinctID: "install_a", sessionID: SESSION },
      });
    });
  });
});
