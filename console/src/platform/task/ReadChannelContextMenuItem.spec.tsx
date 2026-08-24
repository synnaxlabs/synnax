// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Menu } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskFormWithClient } from "@/platform/task/testutil";
import { awaitTextEditing, commitTextEdit } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

describe("ReadChannelContextMenuItem", () => {
  it("should rename the selected channel through the form", async () => {
    const { form } = await renderInTaskFormWithClient(
      <>
        <Task.ChannelName
          channel={0}
          namePath="config.name"
          id={Task.getChannelNameID("ch1")}
        />
        <Menu.Menu>
          <Task.ReadChannelContextMenuItem keys={["ch1"]} channels={[]} />
        </Menu.Menu>
      </>,
      { client, values: { config: { name: "name_before" } } },
    );
    fireEvent.click(await screen.findByText("Rename"));
    const el = await awaitTextEditing(Task.getChannelNameID("ch1"));
    act(() => commitTextEdit(el, "name_after"));
    await waitFor(() =>
      expect(form.current?.get("config.name").value).toBe("name_after"),
    );
  });

  it("should render nothing when multiple channels are selected", async () => {
    await renderInTaskFormWithClient(
      <Menu.Menu>
        <Task.ReadChannelContextMenuItem keys={["ch1", "ch2"]} channels={[]} />
      </Menu.Menu>,
      { client },
    );
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("should render nothing for a viewer, who cannot rename a channel", async () => {
    await renderInTaskFormWithClient(
      <Menu.Menu>
        <Task.ReadChannelContextMenuItem keys={["ch1"]} channels={[]} />
      </Menu.Menu>,
      { client: await roles.get("Viewer") },
    );
    expect(screen.queryByText("Rename")).toBeNull();
  });
});
