// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, createTestClient, DataType, panel } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { Session } from "@/session";
import { assertDefined, renderLinkHook, waitForFocusedTab } from "@/testutil";

const client = createTestClient();

describe("Channel.useLink", () => {
  it("should create a line plot for the channel and open it as a tab", async () => {
    const { layout: _, ...project } = await client.projects.create({
      name: id.create(),
      layout: {},
    });
    const ch = await client.channels.create({
      name: channel.escapeInvalidName(`ch-${id.create()}`),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const { handler, store } = await renderLinkHook(Channel.useLink, { client });
    store.dispatch(Session.Project.select(project.key));
    await handler({ client, key: String(ch.key) });
    const focused = await waitForFocusedTab(store);
    const panelKey = Session.Panel.selectSelected(store.getState());
    assertDefined(panelKey);
    const doc = await client.panels.retrieve(panelKey);
    const tab = panel.findTab(doc.root, focused);
    if (tab?.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe("lineplot");
    const plot = await client.lineplots.retrieve({ key: tab.resource.key });
    expect(plot.name).toBe(`${ch.name} Plot`);
    expect(plot.channels.y1).toEqual([ch.key]);
  });
});
