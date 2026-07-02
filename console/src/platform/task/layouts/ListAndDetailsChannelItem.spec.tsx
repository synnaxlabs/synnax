// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/pluto";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";
import { type Channel } from "@/platform/task/types";

// The channel item renders inside its real Select/List context by being handed to the
// layouts List as its listItem render prop.
const renderItem = (
  extra: Partial<Parameters<typeof Task.Layouts.ListAndDetailsChannelItem>[0]> = {},
) =>
  renderInTaskForm(
    <Task.Layouts.List<Channel>
      createChannel={() => null}
      listItem={Component.renderProp((p) => (
        <Task.Layouts.ListAndDetailsChannelItem
          {...p}
          port="AI0"
          portMaxChars={4}
          canTare
          channel={0}
          path={`config.channels.${p.itemKey}`}
          hasTareButton
          {...extra}
        />
      ))}
    />,
    { values: { config: { channels: [{ key: "a", enabled: true }] } } },
  );

describe("layouts.ListAndDetailsChannelItem", () => {
  it("should render the port label", async () => {
    await renderItem();
    await waitFor(() => expect(screen.getByText("AI0")).toBeTruthy());
  });

  it("should render the default channel name when no channel is set", async () => {
    await renderItem();
    await waitFor(() => expect(screen.getByText("No channel")).toBeTruthy());
  });

  it("should render a tare button when hasTareButton is set", async () => {
    const { container } = await renderItem();
    await waitFor(() =>
      expect(container.querySelector('[aria-label="pluto-icon--tare"]')).toBeTruthy(),
    );
  });

  it("should not render a tare button when hasTareButton is false", async () => {
    const { container } = await renderItem({ hasTareButton: false });
    await waitFor(() => expect(screen.getByText("AI0")).toBeTruthy());
    expect(container.querySelector('[aria-label="pluto-icon--tare"]')).toBeNull();
  });
});
