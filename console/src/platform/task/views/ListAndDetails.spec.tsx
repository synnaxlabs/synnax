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
import { getIconButton } from "@/testutil";

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

const renderListAndDetails = (
  channels: Channel[],
  createChannel: Task.Views.CreateChannel<Channel> = () => null,
) =>
  renderInTaskForm(
    <Task.Views.ListAndDetails<Channel>
      createChannel={createChannel}
      listItem={listItem}
      details={details}
    />,
    { values: { config: { channels } } },
  );

describe("layouts.ListAndDetails", () => {
  it("should show an empty state and a disabled details panel when there are no channels", async () => {
    const { container } = await renderListAndDetails([]);
    await waitFor(() => expect(screen.getByText("No channels in task.")).toBeTruthy());
    expect(screen.queryByText(/details-for-/)).toBeNull();
    const copyButton = getIconButton(container, "json");
    expect(copyButton.className).toContain("pluto--disabled");
  });

  it("should switch the details panel to the channel that is selected", async () => {
    await renderListAndDetails([
      { key: "a", enabled: true },
      { key: "b", enabled: true },
    ]);
    fireEvent.click(await screen.findByText("item-a"));
    await waitFor(() =>
      expect(screen.getByText("details-for-config.channels.a")).toBeTruthy(),
    );
    fireEvent.click(screen.getByText("item-b"));
    await waitFor(() =>
      expect(screen.getByText("details-for-config.channels.b")).toBeTruthy(),
    );
    expect(screen.queryByText("details-for-config.channels.a")).toBeNull();
  });

  it("should create a channel with the selected key as the copy source when Add is pressed", async () => {
    const createChannel = vi.fn((_channels: Channel[], copyFrom?: string): Channel => ({
      key: `new_${copyFrom ?? "none"}`,
      enabled: true,
    }));
    const { container, form } = await renderListAndDetails(
      [{ key: "a", enabled: true }],
      createChannel,
    );
    fireEvent.click(await screen.findByText("item-a"));
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(() => expect(createChannel).toHaveBeenCalled());
    expect(createChannel.mock.calls[0][1]).toBe("a");
    await waitFor(() =>
      expect(form.current?.get("config.channels").value).toHaveLength(2),
    );
    expect(screen.getByText("item-new_a")).toBeTruthy();
  });

  it("should duplicate the selected channels through the context menu", async () => {
    let counter = 0;
    const createChannel = (_channels: Channel[], copyFrom?: string): Channel => ({
      key: `dup_${copyFrom}_${counter++}`,
      enabled: true,
    });
    const { form } = await renderListAndDetails(
      [{ key: "a", enabled: true }],
      createChannel,
    );
    fireEvent.contextMenu(await screen.findByText("item-a"));
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() =>
      expect(form.current?.get("config.channels").value).toHaveLength(2),
    );
    expect(screen.getByText("item-dup_a_0")).toBeTruthy();
  });

  it("should remove a channel through the context menu and clear the details panel", async () => {
    const { form } = await renderListAndDetails([{ key: "a", enabled: true }]);
    fireEvent.click(await screen.findByText("item-a"));
    await waitFor(() =>
      expect(screen.getByText("details-for-config.channels.a")).toBeTruthy(),
    );
    fireEvent.contextMenu(screen.getByText("item-a"));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(() =>
      expect(form.current?.get("config.channels").value).toHaveLength(0),
    );
    await waitFor(() => expect(screen.queryByText(/details-for-/)).toBeNull());
  });
});
