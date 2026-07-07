// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, panel } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { Session } from "@/session";
import { assertDefined, renderLinkHook, waitForFocusedTab } from "@/testutil";

const client = createTestClient();

describe("Arc.useLink", () => {
  it("should retrieve the arc and open it as a tab", async () => {
    const { layout: _, ...project } = await client.projects.create({
      name: id.create(),
      layout: {},
    });
    const created = await client.arcs.create({
      name: "Control Sequence",
      mode: "graph",
      graph: { nodes: [], edges: [] },
    });
    const { handler, store } = await renderLinkHook(Arc.useLink, { client });
    store.dispatch(Session.Project.select(project.key));
    await handler({ client, key: created.key });
    const focused = await waitForFocusedTab(store);
    const panelKey = Session.Panel.selectSelected(store.getState());
    assertDefined(panelKey);
    const doc = await client.panels.retrieve(panelKey);
    const tab = panel.findTab(doc.root, focused);
    if (tab?.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe("arc");
    const retrieved = await client.arcs.retrieve({ key: tab.resource.key });
    expect(retrieved.name).toBe("Control Sequence");
  });
});
