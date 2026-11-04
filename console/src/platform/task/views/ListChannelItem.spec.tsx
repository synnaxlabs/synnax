// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { List } from "@synnaxlabs/lyra/list";
import { deep } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";
import { type Channel } from "@/platform/task/types";
import { getIconButton } from "@/testutil";

const path = deep.path<{ config: { channels: Channel[] } }>();

const COLUMNS: List.Column[] = [{ name: "Port", width: "10rem" }];

const listItem = Component.renderProp(
  ({ itemKey, ...p }: Task.ChannelListItemProps) => (
    <Task.Views.ListChannelItem
      {...p}
      itemKey={itemKey}
      path={path.config.channels[itemKey]}
      channel={0}
      defaultName={`dev_${itemKey}`}
    >
      <List.Cell>
        <span>port-{itemKey}</span>
      </List.Cell>
    </Task.Views.ListChannelItem>
  ),
);

const renderList = () =>
  renderInTaskForm(
    <Task.Views.List
      path={path.config.channels}
      createChannel={() => ({ key: "b", disabled: false })}
      resolve={() => null}
      listItem={listItem}
      columns={COLUMNS}
      checkable
    />,
    { values: { config: { channels: [{ key: "a", disabled: false }] } } },
  );

const getItems = () => screen.getAllByRole("option");

describe("layouts.ListChannelItem", () => {
  it("should show a heading for each task column, then the channel's", async () => {
    await renderList();
    expect(await screen.findByText("Port")).toBeTruthy();
    expect(screen.getByText("Channel")).toBeTruthy();
  });

  it("should scroll the headings sideways with the channels", async () => {
    const { container } = await renderList();
    await screen.findByText("port-a");
    const scroll = container.querySelector<HTMLElement>(".lyra-list__scroll")!;
    scroll.scrollLeft = 120;
    fireEvent.scroll(scroll);
    const headings = container.querySelector<HTMLElement>(".lyra-list__headings")!;
    expect(headings.scrollLeft).toBe(120);
  });

  it("should show the name a new channel takes", async () => {
    await renderList();
    expect(await screen.findByText("dev_a")).toBeTruthy();
  });

  it("should not select a channel on a plain click", async () => {
    await renderList();
    fireEvent.click(await screen.findByText("port-a"));
    expect(getItems()[0].getAttribute("aria-selected")).toBe("false");
  });

  it("should select a channel through its checkbox", async () => {
    await renderList();
    await screen.findByText("port-a");
    fireEvent.click(within(getItems()[0]).getByRole("checkbox", { name: "Select" }));
    await waitFor(() =>
      expect(getItems()[0].getAttribute("aria-selected")).toBe("true"),
    );
  });

  it("should not select an added channel", async () => {
    const { container } = await renderList();
    await screen.findByText("port-a");
    fireEvent.click(getIconButton(container, "add"));
    await screen.findByText("port-b");
    expect(getItems().map((item) => item.getAttribute("aria-selected"))).toEqual([
      "false",
      "false",
    ]);
  });
});
