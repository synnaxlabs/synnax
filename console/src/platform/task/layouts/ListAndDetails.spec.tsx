// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Select } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";
import { type Channel } from "@/platform/task/types";

const listItem = Component.renderProp(
  ({ itemKey, ...p }: Task.ChannelListItemProps) => (
    <Select.ListItem itemKey={itemKey} {...p}>
      item-{itemKey}
    </Select.ListItem>
  ),
);

const details = Component.renderProp(({ path }: { path: string }) => (
  <span>details-for-{path}</span>
));

describe("layouts.ListAndDetails", () => {
  it("should render both the channel list and the details header", async () => {
    await renderInTaskForm(
      <Task.Layouts.ListAndDetails<Channel>
        createChannel={() => null}
        listItem={listItem}
        details={details}
      />,
      { values: { config: { channels: [{ key: "a", enabled: true }] } } },
    );
    expect(screen.getByText("Channels")).toBeTruthy();
    expect(screen.getByText("Details")).toBeTruthy();
    await waitFor(() => expect(screen.getByText("item-a")).toBeTruthy());
  });

  it("should render the details panel for the selected channel", async () => {
    await renderInTaskForm(
      <Task.Layouts.ListAndDetails<Channel>
        createChannel={() => null}
        listItem={listItem}
        details={details}
      />,
      { values: { config: { channels: [{ key: "a", enabled: true }] } } },
    );
    fireEvent.click(await screen.findByText("item-a"));
    await waitFor(() =>
      expect(screen.getByText("details-for-config.channels.a")).toBeTruthy(),
    );
  });

  it("should create a channel with the selected key as the copy source when Add is pressed", async () => {
    const createChannel = vi.fn(
      (_channels: Channel[], copyFrom?: string): Channel => ({
        key: `new-${copyFrom ?? "none"}`,
        enabled: true,
      }),
    );
    const { form } = await renderInTaskForm(
      <Task.Layouts.ListAndDetails<Channel>
        createChannel={createChannel}
        listItem={listItem}
        details={details}
      />,
      { values: { config: { channels: [{ key: "a", enabled: true }] } } },
    );
    fireEvent.click(await screen.findByText("item-a"));
    fireEvent.click(screen.getByRole("button", { name: "" }));
    await waitFor(() => expect(createChannel).toHaveBeenCalled());
    expect(createChannel.mock.calls[0][1]).toBe("a");
    await waitFor(() =>
      expect(form.current?.get("config.channels").value).toHaveLength(2),
    );
  });

  it("should duplicate the selected channels through the context menu", async () => {
    let counter = 0;
    const createChannel = (_channels: Channel[], copyFrom?: string): Channel => ({
      key: `dup-${copyFrom}-${counter++}`,
      enabled: true,
    });
    const { form } = await renderInTaskForm(
      <Task.Layouts.ListAndDetails<Channel>
        createChannel={createChannel}
        listItem={listItem}
        details={details}
      />,
      { values: { config: { channels: [{ key: "a", enabled: true }] } } },
    );
    fireEvent.contextMenu(await screen.findByText("item-a"));
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() =>
      expect(form.current?.get("config.channels").value).toHaveLength(2),
    );
  });
});
