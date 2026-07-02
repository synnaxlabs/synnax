// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Form, Select } from "@synnaxlabs/pluto";
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

interface HarnessProps {
  selected?: string[];
  onSelect?: (keys: string[]) => void;
  onDuplicate?: (channels: Channel[], keys: string[]) => void;
  onTare?: (keys: string[], channels: Channel[]) => void;
  allowTare?: (keys: string[], channels: Channel[]) => boolean;
}

const Harness = ({
  selected = [],
  onSelect = vi.fn(),
  onDuplicate,
  onTare,
  allowTare,
}: HarnessProps) => {
  const { data, remove } = Form.useFieldList<string, Channel>("config.channels");
  return (
    <Task.ChannelList<Channel>
      data={data}
      remove={remove}
      path="config.channels"
      header={<div>My Header</div>}
      emptyContent={<div>Nothing here</div>}
      listItem={listItem}
      selected={selected}
      onSelect={onSelect}
      onDuplicate={onDuplicate}
      onTare={onTare}
      allowTare={allowTare}
    />
  );
};
Harness.displayName = "ChannelListHarness";

const twoChannels = {
  config: {
    channels: [
      { key: "a", enabled: true },
      { key: "b", enabled: false },
    ],
  },
};

describe("ChannelList", () => {
  it("should render the header and an item for each channel", async () => {
    await renderInTaskForm(<Harness />, { values: twoChannels });
    expect(screen.getByText("My Header")).toBeTruthy();
    await waitFor(() => expect(screen.getByText("item-a")).toBeTruthy());
    expect(screen.getByText("item-b")).toBeTruthy();
  });

  it("should render the empty content when there are no channels", async () => {
    await renderInTaskForm(<Harness />, { values: { config: { channels: [] } } });
    await waitFor(() => expect(screen.getByText("Nothing here")).toBeTruthy());
  });

  it("should invoke onSelect when an item is clicked", async () => {
    const onSelect = vi.fn();
    await renderInTaskForm(<Harness onSelect={onSelect} />, { values: twoChannels });
    fireEvent.click(await screen.findByText("item-a"));
    await waitFor(() => expect(onSelect).toHaveBeenCalled());
  });

  describe("context menu", () => {
    it("should show remove and disable actions for an enabled channel", async () => {
      await renderInTaskForm(<Harness />, { values: twoChannels });
      fireEvent.contextMenu(await screen.findByText("item-a"));
      await waitFor(() => expect(screen.getByText("Remove")).toBeTruthy());
      expect(screen.getByText("Disable")).toBeTruthy();
    });

    it("should show an enable action for a disabled channel", async () => {
      await renderInTaskForm(<Harness />, { values: twoChannels });
      fireEvent.contextMenu(await screen.findByText("item-b"));
      await waitFor(() => expect(screen.getByText("Enable")).toBeTruthy());
    });

    it("should disable a channel through the menu", async () => {
      const { form } = await renderInTaskForm(<Harness />, { values: twoChannels });
      fireEvent.contextMenu(await screen.findByText("item-a"));
      fireEvent.click(await screen.findByText("Disable"));
      await waitFor(() =>
        expect(form.current?.get("config.channels.a.enabled").value).toBe(false),
      );
    });

    it("should enable a channel through the menu", async () => {
      const { form } = await renderInTaskForm(<Harness />, { values: twoChannels });
      fireEvent.contextMenu(await screen.findByText("item-b"));
      fireEvent.click(await screen.findByText("Enable"));
      await waitFor(() =>
        expect(form.current?.get("config.channels.b.enabled").value).toBe(true),
      );
    });

    it("should offer a Duplicate action and invoke onDuplicate", async () => {
      const onDuplicate = vi.fn();
      await renderInTaskForm(<Harness onDuplicate={onDuplicate} />, {
        values: twoChannels,
      });
      fireEvent.contextMenu(await screen.findByText("item-a"));
      fireEvent.click(await screen.findByText("Duplicate"));
      await waitFor(() => expect(onDuplicate).toHaveBeenCalled());
    });

    it("should offer a Tare action when allowed and invoke onTare", async () => {
      const onTare = vi.fn();
      await renderInTaskForm(<Harness onTare={onTare} allowTare={() => true} />, {
        values: twoChannels,
      });
      fireEvent.contextMenu(await screen.findByText("item-a"));
      fireEvent.click(await screen.findByText("Tare"));
      await waitFor(() => expect(onTare).toHaveBeenCalled());
    });

    it("should not offer a Tare action when disallowed", async () => {
      await renderInTaskForm(<Harness allowTare={() => false} />, {
        values: twoChannels,
      });
      fireEvent.contextMenu(await screen.findByText("item-a"));
      await waitFor(() => expect(screen.getByText("Remove")).toBeTruthy());
      expect(screen.queryByText("Tare")).toBeNull();
    });
  });
});
