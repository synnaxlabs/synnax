// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import { Session } from "@/session";
import { renderLinkHook, resolveFocusedTab } from "@/testutil";

const client = createTestClient();

describe("Schematic.useLink", () => {
  it("should open a tab for the retrieved schematic", async () => {
    const project = await client.projects.create({ name: id.create(), layout: {} });
    const s = await client.schematics.create(project.key, { name: "Pump Schematic" });
    const { handler, store } = await renderLinkHook(Schematic.useLink, { client });
    store.dispatch(Session.Project.select(project.key));
    await handler({ client, key: s.key });
    const focusedTab = (await resolveFocusedTab(store, client)).key;
    const panelKey = Session.Panel.selectSelected(store.getState());
    if (panelKey == null) throw new Error("no panel selected");
    const doc = await client.panels.retrieve(panelKey);
    const tab = panel.findTabByResource(doc.root, schematic.ontologyID(s.key));
    expect(tab?.key).toBe(focusedTab);
  });
});
