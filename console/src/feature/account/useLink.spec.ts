// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { scheduler } from "@synnaxlabs/x";
import { type UnlistenFn } from "@tauri-apps/api/event";
import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { Account } from "@/feature/account";
import { Session } from "@/session";
import { renderHookWithConsole, type TestStore } from "@/testutil";

const LINK = `synnax-desktop://activate?${new URLSearchParams({
  state: "minted",
  key: "a.b.c",
  secret: "shh",
  activation: "act",
  email: "someone@example.com",
  user: "user_a",
  analytics: "on",
}).toString()}`;

const PENDING: Session.Account.SliceState = {
  ...Session.Account.ZERO_SLICE_STATE,
  pending: "minted",
};

const ACTIVATED: license.Info = {
  state: "ok",
  warning: "",
  fingerprint: ["aa"],
  license: undefined,
};

interface Harness {
  activate: Mock<NonNullable<Account.LinkDeps["license"]>["activate"]>;
  store: TestStore;
  openURL: (urls: string[]) => void;
  statuses: () => Status.NotificationSpec[];
}

const setup = async (
  overrides: Partial<Account.LinkDeps> = {},
  account?: Session.Account.SliceState,
): Promise<Harness> => {
  const activate = vi.fn(async () => ACTIVATED);
  let openURL: (urls: string[]) => void = () => {};
  const onOpenURL = vi.fn(async (handler: typeof openURL): Promise<UnlistenFn> => {
    openURL = handler;
    return () => {};
  });
  const deps: Account.LinkDeps = {
    engine: "tauri",
    getCurrentURLs: async () => null,
    onOpenURL,
    license: { activate },
    ...overrides,
  };
  const { result, store } = await renderHookWithConsole(
    () => {
      Account.useLink(deps);
      return Status.useNotifications().statuses;
    },
    {
      client: createTestClient(),
      preloadedState:
        account == null ? undefined : { [Session.Account.SLICE_NAME]: account },
    },
  );
  // The listener registers after the first render settles.
  await waitFor(() => expect(onOpenURL).toHaveBeenCalled());
  return {
    activate,
    store,
    openURL: (urls) => openURL(urls),
    statuses: () => result.current,
  };
};

const failed = (h: Harness): boolean =>
  h.statuses().some((s) => s.message === "Failed to log in");

describe("Account.useLink", () => {
  beforeEach(() => sessionStorage.clear());

  it("should apply the license key and store the link the app asked for", async () => {
    const h = await setup();
    h.store.dispatch(Session.Account.beginLogin("minted"));
    act(() => h.openURL([LINK]));
    await waitFor(() => {
      expect(h.activate).toHaveBeenCalledWith("a.b.c");
      expect(Session.Account.selectSliceState(h.store.getState())).toEqual({
        version: 0,
        link: { secret: "shh", email: "someone@example.com", user: "user_a" },
        analyticsAllowed: true,
      });
    });
    expect(failed(h)).toBe(false);
  });

  it("should refuse a link whose state the app did not mint", async () => {
    const h = await setup();
    h.store.dispatch(Session.Account.beginLogin("other"));
    act(() => h.openURL([LINK]));
    await waitFor(() => expect(failed(h)).toBe(true));
    expect(h.activate).not.toHaveBeenCalled();
    expect(Session.Account.selectSliceState(h.store.getState()).link).toBeUndefined();
  });

  it("should refuse a link when no login is pending", async () => {
    const h = await setup();
    act(() => h.openURL([LINK]));
    await waitFor(() => expect(failed(h)).toBe(true));
    expect(h.activate).not.toHaveBeenCalled();
  });

  it("should keep the machine unlinked when the Core rejects the license key", async () => {
    // Nothing injected, so the test Core answers, and it rejects the key "a.b.c".
    const h = await setup({ license: undefined });
    h.store.dispatch(Session.Account.beginLogin("minted"));
    act(() => h.openURL([LINK]));
    await waitFor(() => expect(failed(h)).toBe(true));
    const account = Session.Account.selectSliceState(h.store.getState());
    expect(account.link).toBeUndefined();
    expect(account.received).toBeUndefined();
  });

  it("should finish a login whose activation a reload cut short", async () => {
    const stalled = vi.fn(async () => await new Promise<license.Info>(() => {}));
    const first = await setup({ license: { activate: stalled } }, PENDING);
    act(() => first.openURL([LINK]));
    await waitFor(() => expect(stalled).toHaveBeenCalledWith("a.b.c"));
    const held = Session.Account.selectSliceState(first.store.getState());
    expect(held.received).toEqual({
      key: "a.b.c",
      secret: "shh",
      email: "someone@example.com",
      user: "user_a",
      analyticsAllowed: true,
    });
    const next = await setup({}, held);
    await waitFor(() => {
      expect(next.activate).toHaveBeenCalledWith("a.b.c");
      expect(Session.Account.selectSliceState(next.store.getState())).toEqual({
        version: 0,
        link: { secret: "shh", email: "someone@example.com", user: "user_a" },
        analyticsAllowed: true,
      });
    });
    expect(failed(next)).toBe(false);
  });

  it("should take the link the app was launched with", async () => {
    const h = await setup({ getCurrentURLs: async () => [LINK] }, PENDING);
    await waitFor(() => expect(h.activate).toHaveBeenCalledWith("a.b.c"));
    await waitFor(() =>
      expect(Session.Account.selectSliceState(h.store.getState()).link).toBeDefined(),
    );
    expect(failed(h)).toBe(false);
  });

  it("should not refuse a finished login link during a later login", async () => {
    const first = await setup({ getCurrentURLs: async () => [LINK] }, PENDING);
    await waitFor(() => expect(first.activate).toHaveBeenCalledWith("a.b.c"));
    // A reload during the next login reads the finished link from Tauri again.
    const getCurrentURLs = vi.fn(async () => [LINK]);
    const next = await setup({ getCurrentURLs }, { ...PENDING, pending: "next" });
    await waitFor(() => expect(getCurrentURLs).toHaveBeenCalled());
    await act(async () => await scheduler.flushTaskQueue());
    expect(failed(next)).toBe(false);
    expect(next.activate).not.toHaveBeenCalled();
  });

  it("should be a no-op when the engine is not tauri", async () => {
    const getCurrentURLs = vi.fn(async () => [LINK]);
    const onOpenURL = vi.fn(async (): Promise<UnlistenFn> => () => {});
    await renderHookWithConsole(
      () => Account.useLink({ engine: "web", getCurrentURLs, onOpenURL }),
      { client: createTestClient() },
    );
    // Async effects start after a task queue flush, so wait one out before asserting.
    await act(async () => await scheduler.flushTaskQueue());
    expect(getCurrentURLs).not.toHaveBeenCalled();
    expect(onOpenURL).not.toHaveBeenCalled();
  });
});
