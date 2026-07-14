// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { PagerDuty } from "@/feature/pagerduty";
import { Session } from "@/session";
import { resolveFocusedTab, stubGeometry, uniqueName } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("PagerDuty Task Commands", () => {
  it("should create an alert draft and open its resource tab from the command", async () => {
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
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const created = await client.tasks.retrieve({ key: tab.resource.key });
    expect(created.type).toBe(PagerDuty.Task.ALERT_TYPE);
  });
});
