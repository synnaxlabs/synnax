// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Synnax as Client } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Menu } from "@synnaxlabs/lyra/menu";
import { deep } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import {
  renderInTaskFormWithClient,
  type RenderInTaskFormWithClientResult,
} from "@/platform/task/testutil";
import { awaitTextEditing, awaitTextEditingElement, commitTextEdit } from "@/testutil";

const path = deep.path<{ config: { cmdName: string; stateName: string } }>();

const channel = {
  key: "ch1",
  disabled: false,
  cmdChannel: 0,
  stateChannel: 0,
  cmdChannelName: "",
  stateChannelName: "",
};

const client = createTestClient();
const roles = new RoleClients(client);

const renderMenuWithNames = async (
  as: Client = client,
): Promise<RenderInTaskFormWithClientResult> =>
  await renderInTaskFormWithClient(
    <>
      <Task.WriteChannelNames
        cmdChannel={0}
        stateChannel={0}
        cmdNamePath={path.config.cmdName}
        stateNamePath={path.config.stateName}
        itemKey="ch1"
      />
      <Menu.Menu>
        {Task.writeChannelContextMenuItems({
          keys: ["ch1"],
          channels: [channel],
          target: null,
        })}
      </Menu.Menu>
    </>,
    {
      client: as,
      values: { config: { cmdName: "cmd_before", stateName: "state_before" } },
    },
  );

const Names = (): ReactElement => (
  <Task.WriteChannelNames
    cmdChannel={0}
    stateChannel={0}
    cmdNamePath={path.config.cmdName}
    stateNamePath={path.config.stateName}
    itemKey="ch1"
  />
);

/** Shows the names twice and opens the menu over the second copy. */
const TwoCopies = (): ReactElement => {
  const [target, setTarget] = useState<HTMLElement | null>(null);
  return (
    <>
      <Names />
      <div ref={setTarget} data-testid="target">
        <Names />
      </div>
      <Menu.Menu>
        {Task.writeChannelContextMenuItems({
          keys: ["ch1"],
          channels: [channel],
          target,
        })}
      </Menu.Menu>
    </>
  );
};

describe("WriteChannelContextMenuItems", () => {
  it("should rename the command channel through the form", async () => {
    const { form } = await renderMenuWithNames();
    fireEvent.click(await screen.findByText("Rename command channel"));
    const el = await awaitTextEditing(Task.getChannelNameID("ch1", "cmd"));
    act(() => commitTextEdit(el, "cmd_after"));
    await waitFor(() =>
      expect(form.current?.get(path.config.cmdName).value).toBe("cmd_after"),
    );
    expect(form.current?.get(path.config.stateName).value).toBe("state_before");
  });

  it("should rename the state channel through the form", async () => {
    const { form } = await renderMenuWithNames();
    fireEvent.click(await screen.findByText("Rename state channel"));
    const el = await awaitTextEditing(Task.getChannelNameID("ch1", "state"));
    act(() => commitTextEdit(el, "state_after"));
    await waitFor(() =>
      expect(form.current?.get(path.config.stateName).value).toBe("state_after"),
    );
    expect(form.current?.get(path.config.cmdName).value).toBe("cmd_before");
  });

  it("should rename the copy in the item the menu opened over", async () => {
    await renderInTaskFormWithClient(<TwoCopies />, {
      client,
      values: { config: { cmdName: "cmd_before", stateName: "state_before" } },
    });
    fireEvent.click(await screen.findByText("Rename command channel"));
    const el = await awaitTextEditingElement();
    expect(screen.getByTestId("target").contains(el)).toBe(true);
  });

  it("should render nothing when no matching channel is selected", async () => {
    await renderInTaskFormWithClient(
      <Menu.Menu>
        {Task.writeChannelContextMenuItems({
          keys: ["missing"],
          channels: [channel],
          target: null,
        })}
      </Menu.Menu>,
      { client },
    );
    expect(screen.queryByText("Rename command channel")).toBeNull();
    expect(screen.queryByText("Rename state channel")).toBeNull();
  });

  it("should render nothing for a viewer, who cannot rename a channel", async () => {
    await renderMenuWithNames(await roles.get("Viewer"));
    expect(screen.queryByText("Rename command channel")).toBeNull();
    expect(screen.queryByText("Rename state channel")).toBeNull();
  });
});
