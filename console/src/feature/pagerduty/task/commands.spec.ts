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
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { PagerDuty } from "@/feature/pagerduty";
import { Session } from "@/session";
import { assertDefined, stubGeometry, uniqueName, waitForFocusedTab } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("PagerDuty palette commands", () => {
  it("should open the alert task view from the create alert task command", async () => {
    const proj = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: PagerDuty.Task.COMMANDS,
      client,
    });
    store.dispatch(Session.Project.select(proj.key));
    await openCommandPalette("Create a PagerDuty");
    await selectCommand("Create a PagerDuty Alert Task");
    const focused = await waitForFocusedTab(store);
    const panelKey = Session.Panel.selectSelected(store.getState());
    assertDefined(panelKey);
    const doc = await client.panels.retrieve(panelKey);
    const tab = panel.findTab(doc.root, focused);
    if (tab?.variant !== "view") throw new Error("expected a view tab");
    expect(tab.type).toBe(PagerDuty.Task.ALERT_TYPE);
  });
});
