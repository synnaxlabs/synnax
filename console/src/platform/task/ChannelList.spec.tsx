// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Component } from "@synnaxlabs/lyra/component";
import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { deep } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import {
  awaitCommand,
  renderInTaskForm,
  renderInTaskFormWithClient,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const path = deep.path<{ config: { channels: Task.Channel[] } }>();

const resolve = () => null;

const listItem = Component.renderProp(
  ({ itemKey, tareable, ...p }: Task.ChannelListItemProps) => (
    <Select.Item itemKey={itemKey} {...p}>
      item-{itemKey}
    </Select.Item>
  ),
);

interface HarnessProps {
  selected?: string[];
  onSelect?: (keys: string[]) => void;
  onDuplicate?: (channels: Task.Channel[], keys: string[]) => void;
  tareable?: boolean;
  ports?: ReactNode;
}

const Harness = ({
  selected = [],
  onSelect = vi.fn(),
  onDuplicate,
  tareable,
  ports,
}: HarnessProps) => {
  const { data, remove } = Form.useFieldList(path.config.channels);
  return (
    <Task.ChannelList<Task.Channel>
      resolve={resolve}
      data={data}
      remove={remove}
      path={path.config.channels}
      header={<div>My Header</div>}
      emptyContent={<div>Nothing here</div>}
      listItem={listItem}
      selected={selected}
      onSelect={onSelect}
      onDuplicate={onDuplicate}
      tareable={tareable}
      ports={ports}
    />
  );
};
Harness.displayName = "ChannelListHarness";

const CHANNEL_A: Task.Channel = { key: "a", disabled: false };
const CHANNEL_B: Task.Channel = { key: "b", disabled: true };

const twoChannels = { config: { channels: [CHANNEL_A, CHANNEL_B] } };

const PORTS: Task.Port.Group[] = [
  {
    key: "ports",
    columns: 2,
    cells: [
      {
        key: "0",
        name: "ai0",
        labels: ["0"],
        row: 1,
        column: 1,
        state: "used",
        channel: "a",
        channels: ["a"],
        tooltip: "ai0: a",
      },
      {
        key: "1",
        name: "ai1",
        labels: ["1"],
        row: 1,
        column: 2,
        state: "free",
        channels: [],
        tooltip: "ai1 is free",
      },
    ],
  },
];

const portGrid = <Task.Port.Grid label="Ports" groups={PORTS} onClick={vi.fn()} />;

describe("ChannelList", () => {
  it("should render the empty content when there are no channels", async () => {
    await renderInTaskForm(<Harness />, { values: { config: { channels: [] } } });
    await waitFor(() => expect(screen.getByText("Nothing here")).toBeTruthy());
  });

  it("should show the list's error under the list", async () => {
    const { form } = await renderInTaskForm(<Harness />, { values: twoChannels });
    act(() =>
      form.current?.setStatus(path.config.channels, {
        key: "config.channels",
        variant: "error",
        message: "List error",
      }),
    );
    expect(await screen.findByText("List error")).toBeTruthy();
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
        expect(form.current?.get(path.config.channels.a.disabled).value).toBe(true),
      );
    });

    it("should enable a channel through the menu", async () => {
      const { form } = await renderInTaskForm(<Harness />, { values: twoChannels });
      fireEvent.contextMenu(await screen.findByText("item-b"));
      fireEvent.click(await screen.findByText("Enable"));
      await waitFor(() =>
        expect(form.current?.get(path.config.channels.b.disabled).value).toBe(false),
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
        expect(form.current?.get(path.config.channels).value).toEqual([CHANNEL_B]),
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

    it("should hide the edit actions in preview", async () => {
      const onDuplicate = vi.fn();
      await renderInTaskForm(<Harness onDuplicate={onDuplicate} />, {
        values: twoChannels,
        mode: "preview",
      });
      fireEvent.contextMenu(await screen.findByText("item-a"));
      await waitFor(() => expect(screen.getByText("Reload Console")).toBeTruthy());
      expect(screen.queryByText("Remove")).toBeNull();
      expect(screen.queryByText("Duplicate")).toBeNull();
      expect(screen.queryByText("Disable")).toBeNull();
      expect(screen.queryByText("Enable")).toBeNull();
    });

    it("should act on the channel on a used port of the ports view", async () => {
      const { form } = await renderInTaskForm(<Harness ports={portGrid} />, {
        values: twoChannels,
      });
      fireEvent.contextMenu(await screen.findByRole("button", { name: "ai0" }));
      fireEvent.click(await screen.findByText("Remove"));
      await waitFor(() =>
        expect(form.current?.get(path.config.channels).value).toEqual([CHANNEL_B]),
      );
    });

    it("should offer no channel actions on a free port of a grid with no menu", async () => {
      await renderInTaskForm(<Harness ports={portGrid} />, { values: twoChannels });
      fireEvent.contextMenu(await screen.findByRole("button", { name: "ai1" }));
      await waitFor(() => expect(screen.getByText("Reload Console")).toBeTruthy());
      expect(screen.queryByText("Remove")).toBeNull();
    });
  });

  describe("tare", () => {
    const client = createTestClient();

    const createChannel = async (dataType: string) =>
      (
        await client.channels.create({
          name: uniqueName("tare"),
          dataType,
          virtual: true,
        })
      ).key;

    const renderTareable = async (
      taskKey: task.Key,
      channels: (Task.Channel & { channel: number })[],
      mode?: "preview",
    ) =>
      await renderInTaskFormWithClient(
        <Harness tareable selected={channels.map(({ key }) => key)} />,
        {
          client,
          mode,
          values: {
            key: taskKey,
            status: { details: { running: true } },
            config: { channels },
          },
        },
      );

    it("should tare the enabled float channels of the selection", async () => {
      const [float, disabledFloat, integer] = await Promise.all([
        createChannel("float32"),
        createChannel("float64"),
        createChannel("uint8"),
      ]);
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const tsk = await rack.createTask({
        name: uniqueName("tsk"),
        type: "opc_read",
        config: {},
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        await renderTareable(tsk.key, [
          { key: "a", disabled: false, channel: float },
          { key: "b", disabled: true, channel: disabledFloat },
          { key: "c", disabled: false, channel: integer },
        ]);
        fireEvent.contextMenu(await screen.findByText("item-a"));
        fireEvent.click(await screen.findByText("Tare"));
        const cmd = await awaitCommand(streamer, tsk.key, "tare");
        expect(cmd.args).toEqual({ keys: [float] });
      } finally {
        streamer.close();
      }
    });

    it("should keep Tare in preview", async () => {
      const float = await createChannel("float32");
      await renderTareable(
        "task-1",
        [{ key: "a", disabled: false, channel: float }],
        "preview",
      );
      fireEvent.contextMenu(await screen.findByText("item-a"));
      expect(await screen.findByText("Tare")).toBeTruthy();
    });

    it("should withdraw Tare when the task stops", async () => {
      const float = await createChannel("float32");
      const { form } = await renderTareable("task-1", [
        { key: "a", disabled: false, channel: float },
      ]);
      fireEvent.contextMenu(await screen.findByText("item-a"));
      expect(await screen.findByText("Tare")).toBeTruthy();
      const running = deep.path<{ status: { details: { running: boolean } } }>().status
        .details.running;
      act(() => form.current?.set(running, false));
      await waitFor(() => expect(screen.queryByText("Tare")).toBeNull());
    });
  });
});
