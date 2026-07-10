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
import { Text } from "@synnaxlabs/pluto";
import { screen, waitFor } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm, renderInTaskFormWithClient } from "@/platform/task/testutil";
import { awaitTextEditing, commitTextEdit, uniqueName } from "@/testutil";

describe("ChannelName", () => {
  it("should render the default name when no channel is selected", async () => {
    await renderInTaskForm(
      <Task.ChannelName channel={0} namePath="config.name" defaultName="No channel" />,
      { values: { config: { name: "" } } },
    );
    await waitFor(() => expect(screen.getByText("No channel")).toBeTruthy());
  });

  it("should prefer the form name over the default when set", async () => {
    await renderInTaskForm(
      <Task.ChannelName channel={0} namePath="config.name" defaultName="No channel" />,
      { values: { config: { name: "Manually Named" } } },
    );
    await waitFor(() => expect(screen.getByText("Manually Named")).toBeTruthy());
  });

  it("should write a committed rename to the form when no channel is bound", async () => {
    const editID = Task.getChannelNameID("unbound");
    const { form } = await renderInTaskForm(
      <Task.ChannelName channel={0} namePath="config.name" id={editID} />,
      { values: { config: { name: "name_before" } } },
    );
    Text.edit(editID);
    const el = await awaitTextEditing(editID);
    act(() => commitTextEdit(el, "name_after"));
    await waitFor(() =>
      expect(form.current?.get("config.name").value).toBe("name_after"),
    );
  });

  describe("with a live client", () => {
    const client = createTestClient();

    const createChannel = async () =>
      await client.channels.create({
        name: uniqueName("chan"),
        dataType: DataType.FLOAT32,
        virtual: true,
      });

    it("should resolve and display the name of a real channel", async () => {
      const ch = await createChannel();
      await renderInTaskFormWithClient(
        <Task.ChannelName channel={ch.key} namePath="name" />,
        { client, values: { name: "" } },
      );
      await waitFor(() => expect(screen.getByText(ch.name)).toBeTruthy());
    });

    it("should rename the channel on the cluster when an edit is committed", async () => {
      const ch = await createChannel();
      const editID = Task.getChannelNameID("live_ch");
      await renderInTaskFormWithClient(
        <Task.ChannelName channel={ch.key} namePath="name" id={editID} />,
        { client, values: { name: "" } },
      );
      await waitFor(() => expect(screen.getByText(ch.name)).toBeTruthy());
      Text.edit(editID);
      const el = await awaitTextEditing(editID);
      const newName = uniqueName("renamed");
      act(() => commitTextEdit(el, newName));
      await waitFor(async () => {
        const renamed = await client.channels.retrieve(ch.key);
        expect(renamed.name).toBe(newName);
      });
    });
  });
});
