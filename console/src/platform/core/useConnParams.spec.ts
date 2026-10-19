// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Core } from "@/platform/core";
import { createCore, createCoreState } from "@/session/core/testutil";
import { renderHookWithConsole } from "@/testutil";

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

  it("should resume from the session cookie on the web", async () => {
    const { result } = await renderParams();
    expect(result.current).toMatchObject({
      host: "example.com",
      port: 9095,
      sessionCookie: true,
    });
    expect(result.current?.password).toBeUndefined();
    expect(result.current?.token).toBeUndefined();
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
    expect(reads).toEqual([{ account: "example.com:9095" }]);
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
