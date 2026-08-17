// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { getSwitch } from "@/platform/modals/testutil";
import { renderToolbar } from "@/platform/tree/menuTestutil";
import {
  assertDefined,
  createTestClientWithReads,
  getIconButton,
  getIconButtons,
  queryIcon,
  renderHookWithConsole,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const renderChannelToolbar = async (): Promise<void> => {
  await renderToolbar(Channel.TOOLBAR.content, {
    client,
    items: Channel.TREE_ITEMS,
  });
  await screen.findByText("Channels");
};

describe("channel/Toolbar", () => {
  it("opens the create channel modal from the toolbar action", async () => {
    await renderChannelToolbar();
    // The calculated-channel action's composite icon also contains an add glyph, so
    // pick the add button that is not the calculated action.
    const create = await waitFor(() => {
      const calculated = getIconButton(document.body, "calculation");
      const btn = getIconButtons(document.body, "add").find((b) => b !== calculated);
      if (btn == null) throw new Error("create channel action not rendered");
      return btn;
    });
    fireEvent.click(create);
    expect(await screen.findByPlaceholderText("Name")).toBeTruthy();
    expect(getSwitch("Is index")).toBeTruthy();
  });

  it("opens the create calculated channel modal from the toolbar action", async () => {
    await renderChannelToolbar();
    fireEvent.click(await waitFor(() => getIconButton(document.body, "calculation")));
    expect(await screen.findByPlaceholderText("Name")).toBeTruthy();
    expect(await screen.findByText("Derivative")).toBeTruthy();
  });
});

describe("channel/Toolbar permissions", () => {
  it("should withhold the create actions from a viewer", async () => {
    await renderToolbar(Channel.TOOLBAR.content, {
      client: await roles.get("Viewer"),
      items: Channel.TREE_ITEMS,
    });
    await screen.findByText("Channels");
    expect(getIconButtons(document.body, "add")).toHaveLength(0);
    expect(queryIcon(document.body, "calculation")).toBeNull();
  });

  it("should hide the toolbar itself from a subject who cannot read channels", async () => {
    const denied = await createTestClientWithReads(client);
    assertDefined(Channel.TOOLBAR.useVisible);
    const { result } = await renderHookWithConsole(Channel.TOOLBAR.useVisible, {
      client: denied,
    });
    await waitFor(() => expect(result.current).toBe(false));
  });
});
