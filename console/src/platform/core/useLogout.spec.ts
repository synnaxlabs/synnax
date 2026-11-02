// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { act, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Core } from "@/platform/core";
import { Session } from "@/session";
import { createCore, createCoreState, keychainAccount } from "@/session/core/testutil";
import { renderHookWithConsole } from "@/testutil";

describe("Core.useLogout", () => {
  it("should end the session and forget the token, password, and selection", async () => {
    const deleted: unknown[] = [];
    mockIPC((cmd, args) => {
      if (cmd === "keychain_delete") deleted.push(args);
      return undefined;
    });
    onTestFinished(clearMocks);
    const client = createTestClient();
    await client.connect();
    const session = client.auth.session;
    if (session == null) throw new Error("expected a session after connecting");
    const core = createCore("Local", { password: "typed" });
    const { result, store } = await renderHookWithConsole(() => Core.useLogout(), {
      client,
      preloadedState: createCoreState([core], core.key),
    });
    act(() => result.current());
    const state = store.getState();
    expect(Session.Core.selectSelectedKey(state)).toBeUndefined();
    expect(Session.Core.selectState(state, core.key)?.password).toBe("");
    await waitFor(() => expect(client.auth.authenticated).toBe(false));
    await waitFor(() => expect(deleted).toEqual([{ account: keychainAccount(core) }]));
    const other = createTestClient();
    const keys = (await other.auth.retrieveSessions()).map(({ key }) => key);
    expect(keys).not.toContain(session.key);
  });

  it("should forget the token when the Core cannot end the session", async () => {
    const deleted: unknown[] = [];
    mockIPC((cmd, args) => {
      if (cmd === "keychain_delete") deleted.push(args);
      return undefined;
    });
    onTestFinished(clearMocks);
    const client = createTestClient();
    await client.connect();
    vi.spyOn(client.auth, "logout").mockRejectedValueOnce(new Error("unreachable"));
    const core = createCore("Local", { password: "typed" });
    const { result } = await renderHookWithConsole(() => Core.useLogout(), {
      client,
      preloadedState: createCoreState([core], core.key),
    });
    act(() => result.current());
    await waitFor(() => expect(deleted).toEqual([{ account: keychainAccount(core) }]));
  });

  it("should forget the token before the Core ends the session", async () => {
    const deleted: unknown[] = [];
    mockIPC((cmd, args) => {
      if (cmd === "keychain_delete") deleted.push(args);
      return undefined;
    });
    onTestFinished(clearMocks);
    const client = createTestClient();
    await client.connect();
    onTestFinished(async () => await client.close());
    vi.spyOn(client.auth, "logout").mockReturnValueOnce(new Promise(() => {}));
    const core = createCore("Local", { password: "typed" });
    const { result } = await renderHookWithConsole(() => Core.useLogout(), {
      client,
      preloadedState: createCoreState([core], core.key),
    });
    act(() => result.current());
    await waitFor(() => expect(deleted).toEqual([{ account: keychainAccount(core) }]));
  });
});
