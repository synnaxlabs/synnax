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

const listItem = Component.renderProp(({ itemKey }: { itemKey: string }) => (
  <span>item-{itemKey}</span>
));

describe("layouts.List", () => {
  it("should render the Channels header and empty content when there are no channels", async () => {
    await renderInTaskForm(
      <Task.Layouts.List<Channel> createChannel={() => null} listItem={listItem} />,
      { values: { config: { channels: [] } } },
    );
    expect(screen.getByText("Channels")).toBeTruthy();
    await waitFor(() => expect(screen.getByText("No channels in task.")).toBeTruthy());
  });

  it("should render an item for each seeded channel", async () => {
    await renderInTaskForm(
      <Task.Layouts.List<Channel> createChannel={() => null} listItem={listItem} />,
      { values: { config: { channels: [{ key: "a", enabled: true }] } } },
    );
    await waitFor(() => expect(screen.getByText("item-a")).toBeTruthy());
  });

  it("should append a channel to the form when Add is pressed", async () => {
    const createChannel = vi.fn((): Channel => ({ key: "new", enabled: true }));
    const { form } = await renderInTaskForm(
      <Task.Layouts.List<Channel> createChannel={createChannel} listItem={listItem} />,
      { values: { config: { channels: [] } } },
    );
    fireEvent.click(screen.getByRole("button", { name: "" }));
    expect(createChannel).toHaveBeenCalled();
    await waitFor(() =>
      expect(form.current?.get("config.channels").value).toHaveLength(1),
    );
  });
});
