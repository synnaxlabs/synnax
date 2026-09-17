// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Component } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import {
  createTestChannel,
  renderInTaskFormWithClient,
} from "@/platform/task/testutil";
import { type Channel } from "@/platform/task/types";
import {
  getIconButton,
  getToggleButton,
  isToggled,
  queryIcon,
  queryIconButton,
} from "@/testutil";

// The channel item renders inside its real Select/List context by being handed to the
// layouts List as its listItem render prop.
const renderItem = (
  extra: Partial<Parameters<typeof Task.Views.ListAndDetailsChannelItem>[0]> = {},
  client: Synnax | null = null,
) =>
  renderInTaskFormWithClient(
    <Task.Views.List<Channel>
      createChannel={() => null}
      listItem={Component.renderProp((p) => (
        <Task.Views.ListAndDetailsChannelItem
          {...p}
          port="AI0"
          portMaxChars={4}
          canTare
          channel={12}
          device={undefined}
          resolve={() => 0}
          path={`config.channels.${p.itemKey}`}
          hasTareButton
          {...extra}
        />
      ))}
    />,
    {
      client,
      values: {
        config: {
          channels: [
            { key: "a", disabled: false, channel: 0, cmdChannel: 0, stateChannel: 0 },
          ],
        },
      },
    },
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

  it("should toggle the channel's disabled flag in the form", async () => {
    const { container, form } = await renderItem({ hasTareButton: false });
    const toggle = await waitFor(() => getToggleButton(container));
    expect(isToggled(toggle)).toBe(true);
    fireEvent.click(toggle);
    await waitFor(() =>
      expect(form.current?.get("config.channels.a.disabled").value).toBe(true),
    );
    expect(isToggled(getToggleButton(container))).toBe(false);
    fireEvent.click(toggle);
    await waitFor(() =>
      expect(form.current?.get("config.channels.a.disabled").value).toBe(false),
    );
    expect(isToggled(getToggleButton(container))).toBe(true);
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

  describe("resolving from the device map", () => {
    const client = createTestClient();

    it("should show the channel the resolver picks for a read row", async () => {
      const ch = await createTestChannel(client);
      await renderItem({ channel: 0, device: {}, resolve: () => ch.key }, client);
      await screen.findByText(ch.name);
    });

    it("should split a resolved pair across a write row's names", async () => {
      const command = await createTestChannel(client, "cmd");
      const state = await createTestChannel(client, "state");
      await renderItem(
        {
          channel: 0,
          stateChannel: 0,
          device: {},
          resolve: () => ({ command: command.key, state: state.key }),
        },
        client,
      );
      await screen.findByText(command.name);
      await screen.findByText(state.name);
    });

    it("should treat a resolved key as the command of a write row", async () => {
      const command = await createTestChannel(client, "cmd");
      await renderItem(
        { channel: 0, stateChannel: 0, device: {}, resolve: () => command.key },
        client,
      );
      await screen.findByText(command.name);
      await screen.findByText("No state channel");
    });

    it("should take the command of a resolved pair for a read row", async () => {
      const command = await createTestChannel(client, "cmd");
      const state = await createTestChannel(client, "state");
      await renderItem(
        {
          channel: 0,
          device: {},
          resolve: () => ({ command: command.key, state: state.key }),
        },
        client,
      );
      await screen.findByText(command.name);
      expect(screen.queryByText(state.name)).toBeNull();
    });

    it("should keep the row's channel while the device is undefined", async () => {
      const ch = await createTestChannel(client);
      await renderItem(
        { channel: ch.key, device: undefined, resolve: () => 0 },
        client,
      );
      await screen.findByText(ch.name);
    });

    it("should bind a read row's channel field in the form", async () => {
      const ch = await createTestChannel(client);
      const { form } = await renderItem(
        { channel: 0, device: {}, resolve: () => ch.key },
        client,
      );
      await waitFor(() =>
        expect(form.current?.get("config.channels.a.channel").value).toBe(ch.key),
      );
    });

    it("should bind a write row's command and state fields in the form", async () => {
      const command = await createTestChannel(client, "cmd");
      const state = await createTestChannel(client, "state");
      const { form } = await renderItem(
        {
          channel: 0,
          stateChannel: 0,
          device: {},
          resolve: () => ({ command: command.key, state: state.key }),
        },
        client,
      );
      await waitFor(() => {
        expect(form.current?.get("config.channels.a.cmdChannel").value).toBe(
          command.key,
        );
        expect(form.current?.get("config.channels.a.stateChannel").value).toBe(
          state.key,
        );
      });
    });

    it("should show no channel when the resolver finds none for a bound read row", async () => {
      const stale = await createTestChannel(client);
      await renderItem({ channel: stale.key, device: {}, resolve: () => 0 }, client);
      await screen.findByText("No channel");
      expect(screen.queryByText(stale.name)).toBeNull();
    });
  });
});
