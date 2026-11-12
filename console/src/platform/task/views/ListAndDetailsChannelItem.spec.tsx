// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel as clientChannel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Component } from "@synnaxlabs/lyra/component";
import { deep } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm, renderInTaskFormWithClient } from "@/platform/task/testutil";
import { type Channel } from "@/platform/task/types";
import { queryIcon, queryIconButton, uniqueName } from "@/testutil";

const path = deep.path<{ config: { channels: Channel[] } }>();

const client = createTestClient();

const createChannel = async (dataType: string): Promise<clientChannel.Channel> =>
  await client.channels.create({ name: uniqueName("tare"), dataType, virtual: true });

const [FLOAT, INTEGER] = await Promise.all([
  createChannel("float32"),
  createChannel("uint8"),
]);

interface RenderTareableOptions {
  tareable?: boolean;
  snapshot?: boolean;
}

// Renders the item for an existing channel in a running task. The channel's name shows
// once the item has retrieved it.
const renderTareable = async (
  ch: clientChannel.Channel,
  { tareable = true, snapshot = false }: RenderTareableOptions = {},
) => {
  const result = await renderInTaskFormWithClient(
    <Task.Views.List<Channel>
      path={path.config.channels}
      createChannel={() => null}
      resolve={() => null}
      tareable={tareable}
      listItem={Component.renderProp((p) => (
        <Task.Views.ListAndDetailsChannelItem
          {...p}
          port="AI0"
          portMaxChars={4}
          channel={ch.key}
          path={path.config.channels[p.itemKey]}
        />
      ))}
    />,
    {
      client,
      values: {
        key: "task-1",
        snapshot,
        status: { details: { running: true } },
        config: { channels: [{ key: "a", disabled: false }] },
      },
    },
  );
  expect(await screen.findByText(ch.name)).toBeTruthy();
  return result;
};

// The channel item renders inside its real Select/List context by being handed to the
// layouts List as its listItem render prop.
const renderItem = (
  extra: Partial<Parameters<typeof Task.Views.ListAndDetailsChannelItem>[0]> = {},
) =>
  renderInTaskForm(
    <Task.Views.List<Channel>
      path={path.config.channels}
      createChannel={() => null}
      resolve={() => null}
      listItem={Component.renderProp((p) => (
        <Task.Views.ListAndDetailsChannelItem
          {...p}
          port="AI0"
          portMaxChars={4}
          channel={12}
          path={path.config.channels[p.itemKey]}
          {...extra}
        />
      ))}
    />,
    { values: { config: { channels: [{ key: "a", disabled: false }] } } },
  );

describe("layouts.ListAndDetailsChannelItem", () => {
  it("should show a tare button for a float channel of a tareable list", async () => {
    const { container } = await renderTareable(FLOAT);
    await waitFor(() => expect(queryIconButton(container, "tare")).toBeTruthy());
  });

  it("should omit the tare button for a channel that is not a float", async () => {
    const { container } = await renderTareable(INTEGER);
    await act(async () => {});
    expect(queryIcon(container, "tare")).toBeNull();
  });

  it("should omit the tare button from a list that is not tareable", async () => {
    const { container } = await renderTareable(FLOAT, { tareable: false });
    await act(async () => {});
    expect(queryIcon(container, "tare")).toBeNull();
  });

  it("should omit the tare button from a snapshot", async () => {
    const { container } = await renderTareable(FLOAT, { snapshot: true });
    await act(async () => {});
    expect(queryIcon(container, "tare")).toBeNull();
  });

  it("should toggle the channel's disabled flag in the form", async () => {
    const { container, form } = await renderItem();
    const enabled = (): HTMLInputElement =>
      within(container).getByRole<HTMLInputElement>("checkbox");
    await waitFor(() => expect(enabled().checked).toBe(true));
    fireEvent.click(enabled());
    await waitFor(() =>
      expect(form.current?.get(path.config.channels.a.disabled).value).toBe(true),
    );
    expect(enabled().checked).toBe(false);
    fireEvent.click(enabled());
    await waitFor(() =>
      expect(form.current?.get(path.config.channels.a.disabled).value).toBe(false),
    );
    expect(enabled().checked).toBe(true);
  });

  it("should render command and state names when a state channel is present", async () => {
    await renderItem({ stateChannel: 13 });
    await waitFor(() => expect(screen.getAllByText("No channel")).toHaveLength(2));
  });

  it("should render a single channel name when no state channel is present", async () => {
    await renderItem();
    await waitFor(() => expect(screen.getAllByText("No channel")).toHaveLength(1));
  });

  it("should show the name deploying gives a channel that does not exist yet", async () => {
    await renderItem({ channel: 0, defaultName: "dev1_ai_0" });
    await waitFor(() => expect(screen.getByText("dev1_ai_0")).toBeTruthy());
    expect(screen.queryByText("No channel")).toBeNull();
  });

  it("should show the names deploying gives the command and state channels", async () => {
    await renderItem({
      channel: 0,
      stateChannel: 0,
      defaultName: "dev1_ao_0_cmd",
      stateDefaultName: "dev1_ao_0_state",
    });
    await waitFor(() => expect(screen.getByText("dev1_ao_0_cmd")).toBeTruthy());
    expect(screen.getByText("dev1_ao_0_state")).toBeTruthy();
    expect(screen.queryByText("No channel")).toBeNull();
  });

  it("should show no channel names when the item has no channel", async () => {
    await renderItem({ channel: undefined, defaultName: "dev1_ao_0" });
    await waitFor(() => expect(screen.getByText("AI0")).toBeTruthy());
    expect(screen.queryByText("No channel")).toBeNull();
    expect(screen.queryByText("dev1_ao_0")).toBeNull();
  });
});
