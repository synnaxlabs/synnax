// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished } from "vitest";

import { Settings } from "@/app/settings";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { Session } from "@/session";

const client = createTestClient();

const settingsTabs = (root: panel.Node): panel.Tab[] => {
  if (root.variant !== "leaf") throw new Error("expected a single-leaf root");
  return root.tabs.filter((tab) => tab.variant === "view" && tab.type === "settings");
};

describe("Settings.useOpen", () => {
  it("should open one Settings tab in a draft, however often it runs", async () => {
    const me = (await client.users.current.retrieve()).key;
    const shared = await createServerPanel(client, {
      variant: "leaf",
      tabs: [{ variant: "view", key: uuid.create(), type: "other" }],
    });
    const proj = shared.project;
    if (proj == null) throw new Error("expected a project");
    const { wrapper, store } = await createPanelWrapper({
      client,
      project: proj,
      panelKey: shared.key,
    });
    await primePanel(wrapper, shared.key);
    // Stands in for the strip, which keeps the list live.
    const visible = client.panels.watch({ where: panel.visibleTo(proj, me) });
    onTestFinished(visible.onChange(() => {}));
    await visible.retrieve();
    const drafts = async () =>
      await client.panels.retrieve({ projects: [proj], owners: [me] });
    const { result } = renderHook(() => Settings.useOpen(), { wrapper });
    await act(async () => result.current());
    await waitFor(async () => expect(await drafts()).toHaveLength(1));
    await act(async () => result.current());
    const [draft] = await drafts();
    expect(await drafts()).toHaveLength(1);
    expect(draft).toMatchObject({ name: "Settings", icon: "gear" });
    expect(settingsTabs(draft.root)).toHaveLength(1);
    expect(Session.Panel.selectSelected(store.getState())).toBe(draft.key);
    expect(settingsTabs((await client.panels.retrieve(shared.key)).root)).toEqual([]);
  });
});
