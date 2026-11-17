// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Component } from "@synnaxlabs/lyra/component";
import { Form } from "@synnaxlabs/lyra/form";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { deep } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskFormWithClient } from "@/platform/task/testutil";
import { awaitTextEditing, commitTextEdit, openContextMenu } from "@/testutil";

const path = deep.path<{ config: { name: string } }>();
const channelsPath = deep.path<{ config: { channels: Task.Channel[] } }>().config
  .channels;

const NamedItem = ({ itemKey, ...rest }: Task.ChannelListItemProps) => (
  <Select.Item itemKey={itemKey} {...rest}>
    <Text.Editable
      id={Task.getChannelNameID(itemKey)}
      value="Listed"
      onChange={() => {}}
    />
  </Select.Item>
);

const namedItem = Component.renderProp(NamedItem);

const Channels = () => {
  const { data, remove } = Form.useFieldList(channelsPath);
  return (
    <Task.ChannelList<Task.Channel>
      resolve={null}
      data={data}
      remove={remove}
      path={channelsPath}
      header={null}
      emptyContent={<div />}
      listItem={namedItem}
      selected={[]}
      onSelect={() => {}}
      contextMenuItems={Task.readChannelContextMenuItem}
    />
  );
};

const client = createTestClient();
const roles = new RoleClients(client);

describe("ReadChannelContextMenuItem", () => {
  it("should rename the selected channel through the form", async () => {
    const { form } = await renderInTaskFormWithClient(
      <>
        <Task.ChannelName
          channel={0}
          namePath={path.config.name}
          id={Task.getChannelNameID("ch1")}
        />
        <Menu.Menu>
          {Task.readChannelContextMenuItem({
            keys: ["ch1"],
            channels: [],
            target: null,
          })}
        </Menu.Menu>
      </>,
      { client, values: { config: { name: "name_before" } } },
    );
    fireEvent.click(await screen.findByText("Rename"));
    const el = await awaitTextEditing(Task.getChannelNameID("ch1"));
    act(() => commitTextEdit(el, "name_after"));
    await waitFor(() =>
      expect(form.current?.get(path.config.name).value).toBe("name_after"),
    );
  });

  it("should rename the copy in the channel that was right-clicked", async () => {
    await renderInTaskFormWithClient(
      <>
        <Text.Editable
          id={Task.getChannelNameID("a")}
          value="Elsewhere"
          onChange={() => {}}
        />
        <Channels />
      </>,
      { client, values: { config: { channels: [{ key: "a", disabled: false }] } } },
    );
    await openContextMenu("Listed");
    fireEvent.click(await screen.findByText("Rename"));
    await waitFor(() =>
      expect(screen.getByText("Listed").getAttribute("contenteditable")).toBe("true"),
    );
    expect(screen.getByText("Elsewhere").getAttribute("contenteditable")).not.toBe(
      "true",
    );
  });

  it("should render nothing when multiple channels are selected", async () => {
    await renderInTaskFormWithClient(
      <Menu.Menu>
        {Task.readChannelContextMenuItem({
          keys: ["ch1", "ch2"],
          channels: [],
          target: null,
        })}
      </Menu.Menu>,
      { client },
    );
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("should render nothing for a viewer, who cannot rename a channel", async () => {
    await renderInTaskFormWithClient(
      <Menu.Menu>
        {Task.readChannelContextMenuItem({ keys: ["ch1"], channels: [], target: null })}
      </Menu.Menu>,
      { client: await roles.get("Viewer") },
    );
    expect(screen.queryByText("Rename")).toBeNull();
  });
});
