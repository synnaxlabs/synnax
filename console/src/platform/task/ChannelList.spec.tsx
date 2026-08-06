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
  onDuplicate?: (channels: Task.Channel[], keys: string[]) => void;
  onTare?: (keys: string[], channels: Task.Channel[]) => void;
  allowTare?: (keys: string[], channels: Task.Channel[]) => boolean;
}

const Harness = ({
  selected = [],
  onSelect = vi.fn(),
  onDuplicate,
  onTare,
  allowTare,
}: HarnessProps) => {
  const { data, remove } = Form.useFieldList<string, Task.Channel>("config.channels");
  return (
    <Task.ChannelList<Task.Channel>
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

const CHANNEL_A: Task.Channel = { key: "a", enabled: true };
const CHANNEL_B: Task.Channel = { key: "b", enabled: false };

const twoChannels = { config: { channels: [CHANNEL_A, CHANNEL_B] } };

describe("ChannelList", () => {
  it("should render the empty content when there are no channels", async () => {
    await renderInTaskForm(<Harness />, { values: { config: { channels: [] } } });
    await waitFor(() => expect(screen.getByText("Nothing here")).toBeTruthy());
  });

  it("should select the clicked channel", async () => {
    const onSelect = vi.fn();
    await renderInTaskForm(<Harness onSelect={onSelect} />, { values: twoChannels });
    fireEvent.click(await screen.findByText("item-a"));
    await waitFor(() => expect(onSelect).toHaveBeenCalledWith(["a"]));
  });

  describe("context menu", () => {
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

    it("should remove the channel from the form and reselect the remainder", async () => {
      const onSelect = vi.fn();
      const { form } = await renderInTaskForm(<Harness onSelect={onSelect} />, {
        values: twoChannels,
      });
      fireEvent.contextMenu(await screen.findByText("item-a"));
      fireEvent.click(await screen.findByText("Remove"));
      await waitFor(() =>
        expect(form.current?.get("config.channels").value).toEqual([CHANNEL_B]),
      );
      expect(onSelect).toHaveBeenCalledWith(["b"]);
    });

    it("should pass all channels and the selected keys to onDuplicate", async () => {
      const onDuplicate = vi.fn();
      await renderInTaskForm(<Harness onDuplicate={onDuplicate} />, {
        values: twoChannels,
      });
      fireEvent.contextMenu(await screen.findByText("item-a"));
      fireEvent.click(await screen.findByText("Duplicate"));
      await waitFor(() =>
        expect(onDuplicate).toHaveBeenCalledWith([CHANNEL_A, CHANNEL_B], ["a"]),
      );
    });

    it("should pass the selected keys and channels to onTare when allowed", async () => {
      const onTare = vi.fn();
      await renderInTaskForm(<Harness onTare={onTare} allowTare={() => true} />, {
        values: twoChannels,
      });
      fireEvent.contextMenu(await screen.findByText("item-a"));
      fireEvent.click(await screen.findByText("Tare"));
      await waitFor(() => expect(onTare).toHaveBeenCalledWith(["a"], [CHANNEL_A]));
    });

    it("should not offer a Tare action when disallowed", async () => {
      await renderInTaskForm(<Harness allowTare={() => false} />, {
        values: twoChannels,
      });
      fireEvent.contextMenu(await screen.findByText("item-a"));
      await waitFor(() => expect(screen.getByText("Remove")).toBeTruthy());
      expect(screen.queryByText("Tare")).toBeNull();
    });

    it("should hide all mutating actions for a snapshot task", async () => {
      const onDuplicate = vi.fn();
      await renderInTaskForm(
        <Harness onDuplicate={onDuplicate} onTare={vi.fn()} allowTare={() => true} />,
        { values: { ...twoChannels, snapshot: true } },
      );
      fireEvent.contextMenu(await screen.findByText("item-a"));
      await waitFor(() => expect(screen.getByText("Reload console")).toBeTruthy());
      expect(screen.queryByText("Remove")).toBeNull();
      expect(screen.queryByText("Duplicate")).toBeNull();
      expect(screen.queryByText("Disable")).toBeNull();
      expect(screen.queryByText("Enable")).toBeNull();
      expect(screen.queryByText("Tare")).toBeNull();
    });
  });
});
