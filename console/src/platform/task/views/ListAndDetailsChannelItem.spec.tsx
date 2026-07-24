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
import { getIconButton, queryIcon, queryIconButton } from "@/testutil";

// The channel item renders inside its real Select/List context by being handed to the
// layouts List as its listItem render prop.
const renderItem = (
  extra: Partial<Parameters<typeof Task.Views.ListAndDetailsChannelItem>[0]> = {},
) =>
  renderInTaskForm(
    <Task.Views.List<Channel>
      createChannel={() => null}
      listItem={Component.renderProp((p) => (
        <Task.Views.ListAndDetailsChannelItem
          {...p}
          port="AI0"
          portMaxChars={4}
          canTare
          channel={12}
          path={`config.channels.${p.itemKey}`}
          hasTareButton
          {...extra}
        />
      ))}
    />,
    { values: { config: { channels: [{ key: "a", enabled: true }] } } },
  );

describe("layouts.ListAndDetailsChannelItem", () => {
  it("should invoke onTare with the channel key when the tare button is pressed", async () => {
    const onTare = vi.fn();
    const { container } = await renderItem({ onTare });
    await waitFor(() => expect(queryIconButton(container, "tare")).toBeTruthy());
    fireEvent.click(getIconButton(container, "tare"));
    expect(onTare).toHaveBeenCalledWith(12);
  });

  it("should not tare when canTare is false", async () => {
    const onTare = vi.fn();
    const { container } = await renderItem({ onTare, canTare: false });
    await waitFor(() => expect(queryIconButton(container, "tare")).toBeTruthy());
    fireEvent.click(getIconButton(container, "tare"));
    expect(onTare).not.toHaveBeenCalled();
  });

  it("should omit the tare button when hasTareButton is false", async () => {
    const { container } = await renderItem({ hasTareButton: false });
    await waitFor(() => expect(screen.getByText("AI0")).toBeTruthy());
    expect(queryIcon(container, "tare")).toBeNull();
  });

  it("should toggle the channel's enabled flag in the form", async () => {
    const { container, form } = await renderItem({ hasTareButton: false });
    await waitFor(() => expect(queryIconButton(container, "circle")).toBeTruthy());
    const toggle = getIconButton(container, "circle");
    fireEvent.click(toggle);
    await waitFor(() =>
      expect(form.current?.get("config.channels.a.enabled").value).toBe(false),
    );
    fireEvent.click(toggle);
    await waitFor(() =>
      expect(form.current?.get("config.channels.a.enabled").value).toBe(true),
    );
  });

  it("should render command and state names when a state channel is present", async () => {
    await renderItem({ stateChannel: 13 });
    await waitFor(() => expect(screen.getByText("No command channel")).toBeTruthy());
    expect(screen.getByText("No state channel")).toBeTruthy();
    expect(screen.queryByText("No channel")).toBeNull();
  });

  it("should render a single channel name when no state channel is present", async () => {
    await renderItem();
    await waitFor(() => expect(screen.getByText("No channel")).toBeTruthy());
    expect(screen.queryByText("No command channel")).toBeNull();
  });
});
