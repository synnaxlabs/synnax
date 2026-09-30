// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { type UnlistenFn } from "@tauri-apps/api/event";
import { act, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, type MockInstance, vi } from "vitest";

import { Account } from "@/feature/account";
import { type Link } from "@/platform/link";
import { Session } from "@/session";
import { renderHookWithConsole, type TestStore } from "@/testutil";

const LINKED: Account.Linked = {
  state: "minted",
  key: "a.b.c",
  secret: "shh",
  activation: "act",
  email: "someone@example.com",
  user: "user_a",
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
  activate: MockInstance<Client["license"]["activate"]>;
  store: TestStore;
  openURL: (urls: string[]) => void;
  statuses: () => Status.NotificationSpec[];
}

const setup = async (overrides: Partial<Link.Deps> = {}): Promise<Harness> => {
  const client = createTestClient();
  const activate = vi.spyOn(client.license, "activate").mockResolvedValue(ACTIVATED);
  let openURL: (urls: string[]) => void = () => {};
  const onOpenURL = vi.fn(async (handler: typeof openURL): Promise<UnlistenFn> => {
    openURL = handler;
    return () => {};
  });
  const deps: Link.Deps = {
    engine: "tauri",
    getCurrentURLs: async () => null,
    onOpenURL,
    ...overrides,
  };
  const { result, store } = await renderHookWithConsole(
    () => {
      Account.useLink(deps);
      return Status.useNotifications().statuses;
    },
    { client },
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
  afterEach(() => vi.restoreAllMocks());

  it("should apply the license key and store the link the app asked for", async () => {
    const h = await setup();
    h.store.dispatch(Session.Account.beginLogin("minted"));
    act(() => h.openURL([linkOf(LINKED)]));
    await waitFor(() => {
      expect(h.activate).toHaveBeenCalledWith("a.b.c");
      expect(Session.Account.selectSliceState(h.store.getState())).toEqual({
        version: 0,
        activation: "act",
        secret: "shh",
        email: "someone@example.com",
        user: "user_a",
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
    expect(Session.Account.selectSliceState(h.store.getState()).secret).toBeUndefined();
  });

  it("should refuse a link when no login is pending", async () => {
    const h = await setup();
    act(() => h.openURL([linkOf(LINKED)]));
    await waitFor(() => expect(failed(h)).toBe(true));
    expect(h.activate).not.toHaveBeenCalled();
  });

  it("should keep the machine unlinked when the Core rejects the license key", async () => {
    const h = await setup();
    vi.mocked(h.activate).mockRejectedValue(new Error("bad license key"));
    h.store.dispatch(Session.Account.beginLogin("minted"));
    act(() => h.openURL([linkOf(LINKED)]));
    await waitFor(() => expect(failed(h)).toBe(true));
    expect(Session.Account.selectSliceState(h.store.getState()).secret).toBeUndefined();
  });

  it("should take the link the app was launched with", async () => {
    const h = await setup({ getCurrentURLs: async () => [linkOf(LINKED)] });
    await waitFor(() => expect(failed(h)).toBe(true));
  });
});
