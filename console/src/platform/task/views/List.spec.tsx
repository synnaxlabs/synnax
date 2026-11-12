// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { deep } from "@synnaxlabs/x";
import { fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";
import { type Channel } from "@/platform/task/types";
import { getIconButton, queryIconButton, renderWithConsole } from "@/testutil";

const path = deep.path<{ config: { channels: Channel[] } }>();

const listItem = Component.renderProp(({ itemKey }: { itemKey: string }) => (
  <span>item-{itemKey}</span>
));

const selectItem = Component.renderProp(
  ({ itemKey, ...p }: Task.ChannelListItemProps) => (
    <Select.Item itemKey={itemKey} {...p}>
      item-{itemKey}
    </Select.Item>
  ),
);

const resolve = () => null;

const map = ({ onSelect }: Task.Views.PortsProps<Channel>) => (
  <button type="button" onClick={() => onSelect(["a"])}>
    pick-a
  </button>
);

const renderWithMap = () =>
  renderInTaskForm(
    <Task.Views.List<Channel>
      path={path.config.channels}
      createChannel={() => null}
      listItem={selectItem}
      resolve={resolve}
      map={map}
    />,
    { values: { config: { channels: [{ key: "a", disabled: false }] } } },
  );

describe("layouts.List", () => {
  it("should render an item for each seeded channel", async () => {
    await renderInTaskForm(
      <Task.Views.List<Channel>
        path={path.config.channels}
        createChannel={() => null}
        listItem={listItem}
        resolve={resolve}
      />,
      {
        values: {
          config: {
            channels: [
              { key: "a", disabled: false },
              { key: "b", disabled: true },
            ],
          },
        },
      },
    );
    await waitFor(() => expect(screen.getByText("item-a")).toBeTruthy());
    expect(screen.getByText("item-b")).toBeTruthy();
  });

  it("should append a channel to the form when the header add button is pressed", async () => {
    const createChannel = vi.fn((): Channel => ({ key: "new", disabled: false }));
    const { container, form } = await renderInTaskForm(
      <Task.Views.List<Channel>
        path={path.config.channels}
        createChannel={createChannel}
        listItem={listItem}
        resolve={resolve}
      />,
      { values: { config: { channels: [] } } },
    );
    fireEvent.click(getIconButton(container, "add"));
    expect(createChannel).toHaveBeenCalledWith([], undefined);
    await waitFor(() =>
      expect(form.current?.get(path.config.channels).value).toHaveLength(1),
    );
    await waitFor(() => expect(screen.getByText("item-new")).toBeTruthy());
  });

  it("should copy the selected channel when the add button is pressed", async () => {
    const createChannel = vi.fn((): Channel => ({ key: "new", disabled: false }));
    const channels = [
      { key: "a", disabled: false },
      { key: "b", disabled: false },
    ];
    const { container } = await renderInTaskForm(
      <Task.Views.List<Channel>
        path={path.config.channels}
        createChannel={createChannel}
        listItem={selectItem}
        resolve={resolve}
      />,
      { values: { config: { channels } } },
    );
    fireEvent.click(await screen.findByText("item-b"));
    await waitFor(() =>
      expect(
        screen.getByRole("option", { name: "item-b" }).getAttribute("aria-selected"),
      ).toBe("true"),
    );
    fireEvent.click(getIconButton(container, "add"));
    expect(createChannel).toHaveBeenCalledWith(channels, "b");
  });

  it("should create a channel from the footer action", async () => {
    const createChannel = vi.fn((): Channel => ({ key: "new", disabled: false }));
    const { form } = await renderInTaskForm(
      <Task.Views.List<Channel>
        path={path.config.channels}
        createChannel={createChannel}
        listItem={listItem}
        resolve={resolve}
      />,
      { values: { config: { channels: [] } } },
    );
    await waitFor(() => expect(screen.getByText("No channels in task")).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: "Add channel" }));
    await waitFor(() =>
      expect(form.current?.get(path.config.channels).value).toHaveLength(1),
    );
  });

  it("should read a config without channels as an empty list and write nothing", async () => {
    const schema = z.object({
      config: z.object({
        channels: z
          .array(z.object({ key: z.string(), disabled: z.boolean() }))
          .default([]),
      }),
    });
    const onChange = vi.fn();
    const { result: form } = renderHook(() =>
      Form.use<z.ZodType>({ schema, values: { config: {} }, onChange }),
    );
    await renderWithConsole(
      <Form.Form {...form.current}>
        <Task.Views.List<Channel>
          path={path.config.channels}
          createChannel={() => ({ key: "new", disabled: false })}
          listItem={listItem}
          resolve={resolve}
        />
      </Form.Form>,
    );
    await waitFor(() => expect(screen.getByText("No channels in task")).toBeTruthy());
    expect(onChange).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Add channel" }));
    await waitFor(() => expect(screen.getByText("item-new")).toBeTruthy());
    expect(form.current.get(path.config.channels).value).toEqual([
      { key: "new", disabled: false },
    ]);
  });

  it("should append nothing when createChannel returns null", async () => {
    const { container, form } = await renderInTaskForm(
      <Task.Views.List<Channel>
        path={path.config.channels}
        createChannel={() => null}
        listItem={listItem}
        resolve={resolve}
      />,
      { values: { config: { channels: [] } } },
    );
    fireEvent.click(getIconButton(container, "add"));
    expect(form.current?.get(path.config.channels).value).toHaveLength(0);
    expect(screen.getByText("No channels in task")).toBeTruthy();
  });

  it("should name the list, its empty state, and its add action as told", async () => {
    const { form } = await renderInTaskForm(
      <Task.Views.List<Channel>
        path={path.config.channels}
        createChannel={() => ({ key: "new", disabled: false })}
        listItem={listItem}
        resolve={resolve}
        header={<Task.Views.Header title="Tags" />}
        emptyContent={<span>No tags</span>}
        addLabel="Add tag"
      />,
      { values: { config: { channels: [] } } },
    );
    await screen.findByText("Tags");
    expect(screen.getByText("No tags")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Add tag" }));
    await waitFor(() =>
      expect(form.current?.get(path.config.channels).value).toHaveLength(1),
    );
  });

  it("should offer no duplicate when the list cannot copy a channel", async () => {
    await renderInTaskForm(
      <Task.Views.List<Channel>
        path={path.config.channels}
        createChannel={() => null}
        listItem={selectItem}
        resolve={resolve}
      />,
      { values: { config: { channels: [{ key: "a", disabled: false }] } } },
    );
    fireEvent.contextMenu(await screen.findByText("item-a"));
    await screen.findByText("Remove");
    expect(screen.queryByText("Duplicate")).toBeNull();
  });

  it("should show a map under the title, with no view switch", async () => {
    await renderWithMap();
    expect(await screen.findByText("pick-a")).toBeTruthy();
    expect(screen.getByText("Channels")).toBeTruthy();
    expect(screen.queryByText("Ports")).toBeNull();
  });

  it("should select the channel the map picks", async () => {
    await renderWithMap();
    fireEvent.click(await screen.findByText("pick-a"));
    await waitFor(() =>
      expect(screen.getByRole("option").getAttribute("aria-selected")).toBe("true"),
    );
  });

  it("should hide the add button when the form is in preview", async () => {
    const { container } = await renderInTaskForm(
      <Task.Views.List<Channel>
        path={path.config.channels}
        createChannel={() => null}
        listItem={listItem}
        resolve={resolve}
      />,
      { values: { config: { channels: [] } }, mode: "preview" },
    );
    await waitFor(() => expect(screen.getByText("Channels")).toBeTruthy());
    expect(queryIconButton(container, "add")).toBeNull();
  });
});
