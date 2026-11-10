// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { deep } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm, renderInTaskFormWithClient } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const path = deep.path<{ config: { name: string } }>();

const stateIcon = (name: string): HTMLElement | null =>
  screen.queryByRole("img", { name });

const hoverTip = async (icon: HTMLElement): Promise<string> => {
  fireEvent.pointerOver(icon, { pointerType: "mouse" });
  return (await screen.findByRole("tooltip")).textContent;
};

describe("ChannelNameField", () => {
  it("should mark a channel that deploying will create as new", async () => {
    await renderInTaskForm(
      <Task.ChannelNameField
        channel={0}
        namePath={path.config.name}
        defaultName="dev_value"
      />,
      { values: { config: { name: "" } } },
    );
    const icon = await screen.findByRole("img", { name: "New channel" });
    expect(await hoverTip(icon)).toBe("New channelDeploying the task creates it.");
  });

  it("should not mark a skipped channel as new", async () => {
    await renderInTaskForm(
      <Task.ChannelNameField
        channel={0}
        namePath={path.config.name}
        defaultName="dev_value"
        skipped
      />,
      { values: { config: { name: "" } } },
    );
    await screen.findByPlaceholderText("dev_value");
    expect(stateIcon("New channel")).toBeNull();
  });

  it("should not mark a channel with no name yet as new", async () => {
    await renderInTaskForm(
      <Task.ChannelNameField channel={0} namePath={path.config.name} />,
      { values: { config: { name: "" } } },
    );
    await screen.findByPlaceholderText("Channel name");
    expect(stateIcon("New channel")).toBeNull();
  });

  describe("with a live client", () => {
    const client = createTestClient();

    it("should mark a missing channel and write an edit to the form", async () => {
      const ch = await client.channels.create({
        name: uniqueName("gone"),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await client.channels.delete(ch.key);
      const { form } = await renderInTaskFormWithClient(
        <Task.ChannelNameField
          channel={ch.key}
          namePath={path.config.name}
          defaultName="dev_value"
        />,
        { client, values: { config: { name: "" } } },
      );
      const icon = await screen.findByRole("img", { name: "Channel not found" });
      expect(await hoverTip(icon)).toBe(
        "Channel not foundIt may have been deleted. Deploying the task creates a new " +
          "channel with this name.",
      );
      const input = screen.getByPlaceholderText("dev_value");
      fireEvent.change(input, { target: { value: "renamed" } });
      fireEvent.blur(input);
      await waitFor(() =>
        expect(form.current?.get(path.config.name).value).toBe("renamed"),
      );
    });

    it("should show the name of a channel that exists, with no state icon", async () => {
      const ch = await client.channels.create({
        name: uniqueName("live"),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await renderInTaskFormWithClient(
        <Task.ChannelNameField channel={ch.key} namePath={path.config.name} />,
        { client, values: { config: { name: "" } } },
      );
      await screen.findByDisplayValue(ch.name);
      expect(screen.queryByRole("img")).toBeNull();
    });
  });
});
