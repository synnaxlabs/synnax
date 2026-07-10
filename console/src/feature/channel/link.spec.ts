// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { Session } from "@/session";
import { renderLinkHook } from "@/testutil";

const client = createTestClient();

describe("Channel.useLink", () => {
  it("should create and place a line plot for the retrieved channel", async () => {
    const { layout: _, ...project } = await client.projects.create({
      name: id.create(),
      layout: {},
    });
    const ch = await client.channels.create({
      name: channel.escapeInvalidName(`ch-${id.create()}`),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const { handler, store } = await renderLinkHook(Channel.useLink);
    store.dispatch(Session.Project.select(project.key));
    await handler({ client, key: String(ch.key) });
    const placed = Session.Layout.selectByFilter(
      store.getState(),
      (l) => l.name === `${ch.name} Plot`,
    );
    expect(placed).toBeDefined();
  });
});
