// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { createConsoleWrapper, resolveFocusedTab } from "@/testutil";

const client = createTestClient();

const TAB_TYPE = "test_task_form";
const useOpenTaskTab = Task.createOpenTab(TAB_TYPE);

describe("Task.createOpenTab", () => {
  it("opens a view tab of the factory's type on the cluster", async () => {
    const { wrapper, store } = await createConsoleWrapper({ client });
    const { result } = renderHook(() => useOpenTaskTab(), { wrapper });
    await act(async () => {
      result.current();
    });
    const tab = await resolveFocusedTab(
      store,
      client,
      (t) => t.variant === "view" && t.type === TAB_TYPE,
    );
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.type).toBe(TAB_TYPE);
  });

  it("defaults the tab args to an empty object when invoked with no arguments", async () => {
    const { wrapper, store } = await createConsoleWrapper({ client });
    const { result } = renderHook(() => useOpenTaskTab(), { wrapper });
    await act(async () => {
      result.current();
    });
    const tab = await resolveFocusedTab(
      store,
      client,
      (t) => t.variant === "view" && t.type === TAB_TYPE,
    );
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.args).toEqual({});
  });

  it("carries the provided form args through to the tab", async () => {
    const { wrapper, store } = await createConsoleWrapper({ client });
    const { result } = renderHook(() => useOpenTaskTab(), { wrapper });
    const args = {
      taskKey: uuid.create(),
      deviceKey: uuid.create(),
      rackKey: 3,
      config: { sampleRate: 100 },
    };
    await act(async () => {
      result.current(args);
    });
    const tab = await resolveFocusedTab(
      store,
      client,
      (t) => t.variant === "view" && t.type === TAB_TYPE,
    );
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.args).toEqual(args);
  });

  it("binds each minted hook to its own type", async () => {
    const useOpenA = Task.createOpenTab("test_task_form_a");
    const useOpenB = Task.createOpenTab("test_task_form_b");
    const a = await createConsoleWrapper({ client });
    const b = await createConsoleWrapper({ client });
    const ra = renderHook(() => useOpenA(), { wrapper: a.wrapper });
    const rb = renderHook(() => useOpenB(), { wrapper: b.wrapper });
    await act(async () => {
      ra.result.current();
    });
    await act(async () => {
      rb.result.current();
    });
    const tabA = await resolveFocusedTab(a.store, client, (t) => t.variant === "view");
    const tabB = await resolveFocusedTab(b.store, client, (t) => t.variant === "view");
    if (tabA.variant !== "view" || tabB.variant !== "view")
      throw new Error("expected view tabs");
    expect(tabA.type).toBe("test_task_form_a");
    expect(tabB.type).toBe("test_task_form_b");
  });
});
