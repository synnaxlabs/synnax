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
import { describe, expect, it, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Core } from "@/platform/core";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import { renderHookWithConsole } from "@/testutil";

describe("Core.useLogout", () => {
  it("should clear the selection and forget the token and password", async () => {
    mocks.engine = "tauri";
    const deleted: unknown[] = [];
    mockIPC((cmd, args) => {
      if (cmd === "keychain_delete") deleted.push(args);
      return undefined;
    });
    onTestFinished(clearMocks);
    const core = createCore("Local", { password: "typed" });
    const { result, store } = await renderHookWithConsole(() => Core.useLogout(), {
      preloadedState: createCoreState([core], core.key),
    });
    act(() => result.current());
    const state = store.getState();
    expect(Session.Core.selectSelectedKey(state)).toBeUndefined();
    expect(Session.Core.selectState(state, core.key)?.password).toBe("");
    await waitFor(() => expect(deleted).toEqual([{ account: "localhost:9090" }]));
  });
});
