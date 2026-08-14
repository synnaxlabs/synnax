// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";
import { type Channel } from "@/platform/task/types";
import { getIconButton, queryIconButton } from "@/testutil";

const listItem = Component.renderProp(({ itemKey }: { itemKey: string }) => (
  <span>item-{itemKey}</span>
));

describe("layouts.List", () => {
  it("should render an item for each seeded channel", async () => {
    await renderInTaskForm(
      <Task.Views.List<Channel> createChannel={() => null} listItem={listItem} />,
      {
        values: {
          config: {
            channels: [
              { key: "a", enabled: true },
              { key: "b", enabled: false },
            ],
          },
        },
      },
    );
    await waitFor(() => expect(screen.getByText("item-a")).toBeTruthy());
    expect(screen.getByText("item-b")).toBeTruthy();
  });

  it("should append a channel to the form when the header add button is pressed", async () => {
    const createChannel = vi.fn((): Channel => ({ key: "new", enabled: true }));
    const { container, form } = await renderInTaskForm(
      <Task.Views.List<Channel> createChannel={createChannel} listItem={listItem} />,
      { values: { config: { channels: [] } } },
    );
    fireEvent.click(getIconButton(container, "add"));
    expect(createChannel).toHaveBeenCalledWith([]);
    await waitFor(() =>
      expect(form.current?.get("config.channels").value).toHaveLength(1),
    );
    await waitFor(() => expect(screen.getByText("item-new")).toBeTruthy());
  });

  it("should create a channel from the empty-state action", async () => {
    const createChannel = vi.fn((): Channel => ({ key: "new", enabled: true }));
    const { form } = await renderInTaskForm(
      <Task.Views.List<Channel> createChannel={createChannel} listItem={listItem} />,
      { values: { config: { channels: [] } } },
    );
    await waitFor(() => expect(screen.getByText("No channels in task.")).toBeTruthy());
    fireEvent.click(screen.getByText("Add a channel"));
    await waitFor(() =>
      expect(form.current?.get("config.channels").value).toHaveLength(1),
    );
  });

  it("should append nothing when createChannel returns null", async () => {
    const { container, form } = await renderInTaskForm(
      <Task.Views.List<Channel> createChannel={() => null} listItem={listItem} />,
      { values: { config: { channels: [] } } },
    );
    fireEvent.click(getIconButton(container, "add"));
    expect(form.current?.get("config.channels").value).toHaveLength(0);
    expect(screen.getByText("No channels in task.")).toBeTruthy();
  });

  it("should hide the add button when the task is a snapshot", async () => {
    const { container } = await renderInTaskForm(
      <Task.Views.List<Channel> createChannel={() => null} listItem={listItem} />,
      { values: { snapshot: true, config: { channels: [] } } },
    );
    await waitFor(() => expect(screen.getByText("Channels")).toBeTruthy());
    expect(queryIconButton(container, "add")).toBeNull();
  });
});
