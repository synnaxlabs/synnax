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
import { Cluster } from "@/session/cluster";
import { Nav } from "@/session/nav";
import { Panel } from "@/session/panel";
import { Project } from "@/session/project";
import { renderHookWithConsole } from "@/testutil";

const PROJECT_KEY = "11111111-1111-4111-8111-111111111111";
const PANEL_KEY = "22222222-2222-4222-8222-222222222222";
const TAB_KEY = "33333333-3333-4333-8333-333333333333";

describe("useLogout", () => {
  it("should clear cluster, project, panel session state, and hide nav drawers", async () => {
    const { result, store } = await renderHookWithConsole(() => Session.useLogout());

    act(() => {
      store.dispatch(Cluster.select("LOCAL"));
      store.dispatch(Project.select(PROJECT_KEY));
      store.dispatch(Panel.select({ key: PANEL_KEY, windowKey: MAIN_WINDOW }));
      store.dispatch(
        Panel.internalSelectTab({
          key: PANEL_KEY,
          tabKey: TAB_KEY,
          otherTabKeys: [TAB_KEY],
          windowKey: MAIN_WINDOW,
        }),
      );
      store.dispatch(Nav.selectLeft({ windowKey: MAIN_WINDOW, key: "resources" }));
      store.dispatch(Nav.showBottom({ windowKey: MAIN_WINDOW }));
    });

    const before = store.getState();
    expect(Cluster.selectSelectedKey(before)).toBe("LOCAL");
    expect(Project.selectOptionalSelected(before)).toBe(PROJECT_KEY);
    expect(before[Panel.SLICE_NAME].windows[MAIN_WINDOW].selected).toBe(PANEL_KEY);
    expect(
      before[Panel.SLICE_NAME].windows[MAIN_WINDOW].panels[PANEL_KEY]?.selectedTabs,
    ).toEqual([TAB_KEY]);
    expect(before[Nav.SLICE_NAME].windows[MAIN_WINDOW]?.left.selected).toBe("resources");
    expect(before[Nav.SLICE_NAME].windows[MAIN_WINDOW]?.bottom.visible).toBe(true);

    act(() => {
      result.current();
    });

    const after = store.getState();
    expect(Cluster.selectSelectedKey(after)).toBeUndefined();
    expect(Project.selectOptionalSelected(after)).toBeUndefined();
    expect(Panel.selectSliceState(after)).toEqual(Panel.ZERO_SLICE_STATE);
    expect(after[Nav.SLICE_NAME].windows[MAIN_WINDOW]?.left.selected).toBeUndefined();
    expect(after[Nav.SLICE_NAME].windows[MAIN_WINDOW]?.bottom.visible ?? false).toBe(
      false,
    );
  });

  it("should return a stable callback across renders", async () => {
    const { result, rerender } = await renderHookWithConsole(() => Session.useLogout());
    const first = result.current;
    rerender();
    expect(result.current).toBe(first);
  });
});
