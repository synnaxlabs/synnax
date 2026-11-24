// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Core } from "@/platform/core";
import { Session } from "@/session";
import { createCore, createCoreState, keychainAccount } from "@/session/core/testutil";
import { renderHookWithConsole } from "@/testutil";

const CORE = createCore("Local", { username: "before" });

const renderSetUsername = async (core = CORE) =>
  await renderHookWithConsole(() => Core.useSetUsername(), {
    preloadedState: createCoreState([core], core.key),
  });

describe("Core.useSetUsername", () => {
  beforeEach(() => {
    mocks.engine = "tauri";
  });

  it("should move the saved session to the new username before the record changes", async () => {
    const calls: unknown[] = [];
    const rendered: { store?: Awaited<ReturnType<typeof renderSetUsername>>["store"] } =
      {};
    const recorded = (): string | undefined =>
      rendered.store == null
        ? undefined
        : Session.Core.selectState(rendered.store.getState(), CORE.key)?.username;
    mockIPC((cmd, args) => {
      calls.push({ cmd, args, record: recorded() });
      return cmd === "keychain_get" ? "token" : undefined;
    });
    onTestFinished(clearMocks);
    const { result, store } = await renderSetUsername();
    rendered.store = store;
    act(() => result.current("after"));
    await waitFor(() => expect(recorded()).toBe("after"));
    const renamed = { ...CORE, username: "after" };
    expect(calls).toEqual([
      {
        cmd: "keychain_get",
        args: { account: keychainAccount(CORE) },
        record: "before",
      },
      {
        cmd: "keychain_set",
        args: { account: keychainAccount(renamed), secret: "token" },
        record: "before",
      },
      {
        cmd: "keychain_delete",
        args: { account: keychainAccount(CORE), expected: "token" },
        record: "before",
      },
    ]);
  });

  it("should keep the record when the keychain cannot be read", async () => {
    const calls: string[] = [];
    mockIPC((cmd) => {
      calls.push(cmd);
      if (cmd === "keychain_get") throw new Error("locked");
      return undefined;
    });
    onTestFinished(clearMocks);
    const { result, store } = await renderSetUsername();
    act(() => result.current("after"));
    await waitFor(() => expect(calls).toEqual(["keychain_get"]));
    await act(async () => await Promise.resolve());
    expect(Session.Core.selectState(store.getState(), CORE.key)?.username).toBe(
      "before",
    );
  });

  it("should change only the record on the web", async () => {
    mocks.engine = "web";
    const calls: string[] = [];
    mockIPC((cmd) => {
      calls.push(cmd);
      return undefined;
    });
    onTestFinished(clearMocks);
    const { result, store } = await renderSetUsername();
    act(() => result.current("after"));
    await waitFor(() =>
      expect(Session.Core.selectState(store.getState(), CORE.key)?.username).toBe(
        "after",
      ),
    );
    expect(calls).toEqual([]);
  });
});
