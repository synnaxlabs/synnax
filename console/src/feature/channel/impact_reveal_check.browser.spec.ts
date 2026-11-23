// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel as channelClient, query } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { createReadTask, createVirtualChannel } from "@/feature/channel/testutil";
import { CSS } from "@/platform/css";
import { findSurface } from "@/platform/modals/testutil";
import { Tree } from "@/platform/tree";
import {
  createHostItems,
  HOST_ID,
  openTreeRowContextMenu,
  renderLiveTree,
} from "@/platform/tree/testutil";

const client = createTestClient();

const hostChannels = (...channels: { key: channelClient.Key }[]) =>
  createHostItems((c) => [
    Tree.createSource(
      c.channels,
      query.keysRequest(channels.map((ch) => ch.key)),
      (ch) => Tree.toResource("channel")(ch.payload),
    ),
  ]);

describe("impact dialog reveal check", () => {
  it("should keep focus out of the tree while the dialog waits for its plan", async () => {
    const ch = await createVirtualChannel(client);
    await createReadTask(client, [ch.key]);
    await renderLiveTree({
      client,
      root: HOST_ID,
      items: { ...Channel.TREE_ITEMS, ...hostChannels(ch) },
    });
    const tree = screen.getByRole("tree");
    // A right-click focuses the tree, so its menu gives the focus back to it.
    act(() => tree.focus());
    await openTreeRowContextMenu(ch.name);
    fireEvent.click(await screen.findByText("Delete"));
    // The native dialog shows at once; the surface inside it waits unseen for the plan.
    const surface = findSurface();
    const unseen = CSS.BM("impact", "unpainted");
    const focused: string[] = [];
    let frame = 0;
    for (; frame < 300; frame++) {
      if (tree.contains(document.activeElement)) focused.push(`frame ${frame}`);
      if (!surface.classList.contains(unseen)) break;
      await new Promise((resolve) => requestAnimationFrame(resolve));
    }
    expect(surface.classList.contains(unseen)).toBe(false);
    expect(focused).toEqual([]);
  });
});
