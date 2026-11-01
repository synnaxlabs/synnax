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

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Core } from "@/platform/core";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import { pinLocationOrigin, renderHookWithConsole } from "@/testutil";

const CORE = createCore("Alpha", { host: "example.com", port: 9095, password: "" });

const renderParams = async (core = CORE) =>
  await renderHookWithConsole(() => Core.useConnParams(), {
    preloadedState: createCoreState([core], core.key),
  });

describe("Core.useConnParams", () => {
  beforeEach(() => {
    mocks.engine = "web";
  });

  it("should give no parameters while no Core is selected", async () => {
    const { result } = await renderHookWithConsole(() => Core.useConnParams(), {
      preloadedState: createCoreState([CORE]),
    });
    expect(result.current).toBeUndefined();
  });

  it("should resume from the session cookie on a page the Core serves", async () => {
    pinLocationOrigin("http://example.com:9095");
    const { result } = await renderParams();
    expect(result.current).toMatchObject({
      host: "example.com",
      port: 9095,
      sessionCookie: true,
      sessionPersistent: true,
    });
    expect(result.current?.password).toBeUndefined();
    expect(result.current?.token).toBeUndefined();
  });

  it("should not ask for a cookie from a Core on another origin", async () => {
    pinLocationOrigin("http://localhost:5173");
    const { result } = await renderParams();
    expect(result.current?.sessionCookie).toBe(false);
    expect(result.current?.sessionPersistent).toBe(false);
  });

  it("should ask for a cookie from a secure Core on the page's default port", async () => {
    pinLocationOrigin("https://example.com");
    const { result } = await renderParams({ ...CORE, port: 443, secure: true });
    expect(result.current?.sessionCookie).toBe(true);
  });

  it("should not ask for a cookie when the page and the Core differ in scheme", async () => {
    pinLocationOrigin("https://example.com:9095");
    const { result } = await renderParams();
    expect(result.current?.sessionCookie).toBe(false);
  });

  it("should keep the Core's name and cluster out of the connection", async () => {
    const { result } = await renderParams();
    expect(result.current).not.toHaveProperty("name");
    expect(result.current).not.toHaveProperty("clusterKey");
    expect(result.current).not.toHaveProperty("key");
  });

  it("should log in with a password the user entered", async () => {
    const { result } = await renderParams({ ...CORE, password: "typed" });
    expect(result.current?.password).toBe("typed");
  });

  it("should wait for the keychain and resume from its token on desktop", async () => {
    mocks.engine = "tauri";
    const reads: unknown[] = [];
    mockIPC((cmd, args) => {
      if (cmd !== "keychain_get") return undefined;
      reads.push(args);
      return "sys_saved";
    });
    onTestFinished(clearMocks);
    const { result } = await renderParams();
    expect(result.current).toBeUndefined();
    await waitFor(() => expect(result.current?.token).toBe("sys_saved"));
    expect(result.current?.sessionCookie).toBeUndefined();
    expect(result.current?.sessionPersistent).toBe(true);
    expect(result.current).not.toHaveProperty("name");
    expect(reads).toEqual([{ account: `http://synnax@example.com:9095#${CORE.key}` }]);
  });

  it("should not resume over plain text from a token saved for TLS", async () => {
    mocks.engine = "tauri";
    let reads = 0;
    mockIPC((cmd) => {
      if (cmd !== "keychain_get") return undefined;
      reads += 1;
      return reads === 1 ? "sys_saved" : null;
    });
    onTestFinished(clearMocks);
    const secure = { ...CORE, secure: true };
    const { result, store } = await renderParams(secure);
    await waitFor(() => expect(result.current?.token).toBe("sys_saved"));
    act(() => {
      store.dispatch(Session.Core.set({ ...secure, secure: false }));
    });
    await waitFor(() => expect(result.current?.secure).toBe(false));
    expect(result.current?.token).toBeUndefined();
  });

  it("should keep separate tokens for two saved Cores at one address", async () => {
    mocks.engine = "tauri";
    const alpha = { ...CORE, username: "" };
    const bravo = createCore("Bravo", {
      host: "example.com",
      port: 9095,
      password: "",
      username: "",
    });
    mockIPC((cmd, args) => {
      if (cmd !== "keychain_get") return undefined;
      const { account } = args as { account: string };
      if (account.endsWith(`#${alpha.key}`)) return "sys_alpha";
      if (account.endsWith(`#${bravo.key}`)) return "sys_bravo";
      return null;
    });
    onTestFinished(clearMocks);
    const { result, store } = await renderHookWithConsole(() => Core.useConnParams(), {
      preloadedState: createCoreState([alpha, bravo], alpha.key),
    });
    await waitFor(() => expect(result.current?.token).toBe("sys_alpha"));
    act(() => {
      store.dispatch(Session.Core.select(bravo.key));
    });
    await waitFor(() => expect(result.current?.token).toBe("sys_bravo"));
  });

  it("should connect without a token when the keychain holds none", async () => {
    mocks.engine = "tauri";
    mockIPC((cmd) => (cmd === "keychain_get" ? null : undefined));
    onTestFinished(clearMocks);
    const { result } = await renderParams();
    await waitFor(() => expect(result.current).toBeDefined());
    expect(result.current?.token).toBeUndefined();
  });
});
