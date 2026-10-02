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
import { type UnlistenFn } from "@tauri-apps/api/event";
import { act, waitFor } from "@testing-library/react";
import { describe, expect, it, type Mock, vi } from "vitest";

import { Account } from "@/feature/account";
import { Session } from "@/session";
import { renderHookWithConsole, type TestStore } from "@/testutil";

const LINKED: Account.Linked = {
  state: "minted",
  key: "a.b.c",
  secret: "shh",
  activation: "act",
  email: "someone@example.com",
};

const linkOf = (linked: Account.Linked): string =>
  `${Account.SCHEME}://activate?${new URLSearchParams({ ...linked }).toString()}`;

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

const setup = async (overrides: Partial<Account.LinkDeps> = {}): Promise<Harness> => {
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
    { client: createTestClient() },
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
  it("should apply the license key and store the link the app asked for", async () => {
    const h = await setup();
    h.store.dispatch(Session.Account.beginLogin("minted"));
    act(() => h.openURL([linkOf(LINKED)]));
    await waitFor(() => {
      expect(h.activate).toHaveBeenCalledWith("a.b.c");
      expect(Session.Account.selectSliceState(h.store.getState())).toEqual({
        version: 0,
        link: { secret: "shh", email: "someone@example.com" },
      });
    });
    expect(failed(h)).toBe(false);
  });

  it("should refuse a link whose state the app did not mint", async () => {
    const h = await setup();
    h.store.dispatch(Session.Account.beginLogin("other"));
    act(() => h.openURL([linkOf(LINKED)]));
    await waitFor(() => expect(failed(h)).toBe(true));
    expect(h.activate).not.toHaveBeenCalled();
    expect(Session.Account.selectSliceState(h.store.getState()).link).toBeUndefined();
  });

  it("should refuse a link when no login is pending", async () => {
    const h = await setup();
    act(() => h.openURL([linkOf(LINKED)]));
    await waitFor(() => expect(failed(h)).toBe(true));
    expect(h.activate).not.toHaveBeenCalled();
  });

  it("should keep the machine unlinked when the Core rejects the license key", async () => {
    // Nothing injected, so the test Core answers, and it rejects the key "a.b.c".
    const h = await setup({ license: undefined });
    h.store.dispatch(Session.Account.beginLogin("minted"));
    act(() => h.openURL([linkOf(LINKED)]));
    await waitFor(() => expect(failed(h)).toBe(true));
    expect(Session.Account.selectSliceState(h.store.getState()).link).toBeUndefined();
  });

  it("should take the link the app was launched with", async () => {
    const h = await setup({ getCurrentURLs: async () => [linkOf(LINKED)] });
    await waitFor(() => expect(failed(h)).toBe(true));
  });
});
