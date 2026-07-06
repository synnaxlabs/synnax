// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { act } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Session } from "@/session";
import { Layout } from "@/session/layout";
import { Nav } from "@/session/nav";
import { Node } from "@/session/node";
import { Project } from "@/session/project";
import { renderHookWithConsole } from "@/testutil";

const PROJECT_KEY = "11111111-1111-4111-8111-111111111111";
const MODAL_LAYOUT: Layout.State = {
  windowKey: MAIN_WINDOW,
  key: "logout-modal",
  type: "someModal",
  name: "A Modal",
  location: "modal",
};

describe("useLogout", () => {
  it("should clear cluster, project, layout project state, and hide nav drawers", async () => {
    const { result, store } = await renderHookWithConsole(() => Session.useLogout());

    act(() => {
      store.dispatch(Node.select("LOCAL"));
      store.dispatch(Project.select(PROJECT_KEY));
      store.dispatch(Layout.place(MODAL_LAYOUT));
      store.dispatch(Nav.selectLeft({ windowKey: MAIN_WINDOW, key: "resources" }));
      store.dispatch(Nav.showBottom({ windowKey: MAIN_WINDOW }));
    });

    const before = store.getState();
    expect(Node.selectSelectedKey(before)).toBe("LOCAL");
    expect(Project.selectOptionalSelected(before)).toBe(PROJECT_KEY);
    expect(Layout.select(before, MODAL_LAYOUT.key)).toBeDefined();
    expect(Nav.selectLeftSelected(before)).toBe("resources");
    expect(Nav.selectBottomVisible(before)).toBe(true);

    act(() => {
      result.current();
    });

    const after = store.getState();
    expect(Node.selectSelectedKey(after)).toBeUndefined();
    expect(Project.selectOptionalSelected(after)).toBeUndefined();
    expect(Layout.select(after, MODAL_LAYOUT.key)).toBeUndefined();
    expect(Nav.selectLeftSelected(after)).toBeUndefined();
    expect(Nav.selectBottomVisible(after)).toBe(false);
  });

  it("should return a stable callback across renders", async () => {
    const { result, rerender } = await renderHookWithConsole(() => Session.useLogout());
    const first = result.current;
    rerender();
    expect(result.current).toBe(first);
  });
});
