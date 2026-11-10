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
import { Form } from "@synnaxlabs/lyra/form";
import { Text } from "@synnaxlabs/lyra/text";
import { deep, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm, renderInTaskFormWithClient } from "@/platform/task/testutil";
import { awaitTextEditing, commitTextEdit, uniqueName } from "@/testutil";

const path = deep.path<{ config: { name: string }; name: string; channel: number }>();

const isMissing = (): boolean =>
  screen.queryByRole("img", { name: "Channel not found" }) != null;

const hoverTip = async (name: string): Promise<string> => {
  fireEvent.pointerOver(screen.getByText(name), { pointerType: "mouse" });
  return (await screen.findByRole("tooltip")).textContent;
};

describe("ChannelName", () => {
  it("should show No channel when the channel has no name yet", async () => {
    await renderInTaskForm(
      <Task.ChannelName channel={0} namePath={path.config.name} />,
      { values: { config: { name: "" } } },
    );
    await waitFor(() => expect(screen.getByText("No channel")).toBeTruthy());
    expect(screen.queryByRole("img", { name: "New channel" })).toBeNull();
    expect(await hoverTip("No channel")).toBe(
      "No channelDeploying the task creates and names it.",
    );
  });

  it("should mark a channel that deploying will create as new", async () => {
    await renderInTaskForm(
      <Task.ChannelName
        channel={0}
        namePath={path.config.name}
        defaultName="pressure_0"
      />,
      { values: { config: { name: "" } } },
    );
    await waitFor(() => expect(screen.getByText("pressure_0")).toBeTruthy());
    expect(screen.getByRole("img", { name: "New channel" })).toBeTruthy();
    expect(await hoverTip("pressure_0")).toBe(
      "New channelDeploying the task creates it.",
    );
  });

  it("should mark a named channel with no default as new", async () => {
    await renderInTaskForm(
      <Task.ChannelName channel={0} namePath={path.config.name} />,
      { values: { config: { name: "unbound_name" } } },
    );
    await waitFor(() => expect(screen.getByText("unbound_name")).toBeTruthy());
    expect(screen.getByRole("img", { name: "New channel" })).toBeTruthy();
  });

  it("should lead the name with the given icon", async () => {
    await renderInTaskForm(
      <Task.ChannelName
        channel={0}
        namePath={path.config.name}
        icon={<span>lead</span>}
      />,
      { values: { config: { name: "" } } },
    );
    await waitFor(() => expect(screen.getByText("lead")).toBeTruthy());
  });

  it("should show the new icon in place of the given icon", async () => {
    await renderInTaskForm(
      <Task.ChannelName
        channel={0}
        namePath={path.config.name}
        defaultName="pressure_0"
        icon={<span>lead</span>}
      />,
      { values: { config: { name: "" } } },
    );
    await waitFor(() => expect(screen.getByText("pressure_0")).toBeTruthy());
    expect(screen.queryByText("lead")).toBeNull();
    expect(screen.getByRole("img", { name: "New channel" })).toBeTruthy();
  });

  it("should prefer the form name over the default when set", async () => {
    await renderInTaskForm(
      <Task.ChannelName
        channel={0}
        namePath={path.config.name}
        defaultName="No channel"
      />,
      { values: { config: { name: "Manually Named" } } },
    );
    await waitFor(() => expect(screen.getByText("Manually Named")).toBeTruthy());
  });

  it("should write a committed rename to the form when no channel is bound", async () => {
    const editID = Task.getChannelNameID("unbound");
    const { form } = await renderInTaskForm(
      <Task.ChannelName channel={0} namePath={path.config.name} id={editID} />,
      { values: { config: { name: "name_before" } } },
    );
    act(() => Text.edit(editID));
    const el = await awaitTextEditing(editID);
    act(() => commitTextEdit(el, "name_after"));
    await waitFor(() =>
      expect(form.current?.get(path.config.name).value).toBe("name_after"),
    );
  });

  it("should end the form name in the suffix, and store a rename without it", async () => {
    const editID = Task.getChannelNameID("suffixed");
    const { form } = await renderInTaskForm(
      <Task.ChannelName
        channel={0}
        namePath={path.config.name}
        suffix="_int16"
        id={editID}
      />,
      { values: { config: { name: "pressure" } } },
    );
    await waitFor(() => expect(screen.getByText("pressure_int16")).toBeTruthy());
    act(() => Text.edit(editID));
    const el = await awaitTextEditing(editID);
    act(() => commitTextEdit(el, "flow_int16"));
    await waitFor(() => expect(screen.getByText("flow_int16")).toBeTruthy());
    expect(form.current?.get(path.config.name).value).toBe("flow");
  });

  it("should not mark a skipped channel as new", async () => {
    await renderInTaskForm(
      <Task.ChannelName
        channel={0}
        namePath={path.config.name}
        defaultName="pressure_0"
        icon={<span>lead</span>}
        skipped
      />,
      { values: { config: { name: "" } } },
    );
    await waitFor(() => expect(screen.getByText("pressure_0")).toBeTruthy());
    expect(screen.getByText("lead")).toBeTruthy();
    expect(screen.queryByRole("img", { name: "New channel" })).toBeNull();
    expect(await hoverTip("pressure_0")).toBe(
      "Skipped channelDeploying the task does not create a disabled channel.",
    );
  });

  it("should say why a skipped channel with no name has none", async () => {
    await renderInTaskForm(
      <Task.ChannelName channel={0} namePath={path.config.name} skipped />,
      { values: { config: { name: "" } } },
    );
    expect(await hoverTip("No channel")).toBe(
      "No channelDeploying the task does not create a disabled channel.",
    );
  });

  it("should show No channel and allow no edit while nothing is picked", async () => {
    await renderInTaskForm(<Task.ChannelName channel={0} picked />, {
      values: { config: { name: "" } },
    });
    const name = await screen.findByText("No channel");
    expect(name.className).not.toContain("lyra-text--editable");
    expect(screen.queryByRole("img")).toBeNull();
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
        <Task.ChannelName channel={ch.key} namePath={path.name} />,
        { client, values: { name: "" } },
      );
      await waitFor(() => expect(screen.getByText(ch.name)).toBeTruthy());
      expect(screen.queryByRole("img")).toBeNull();
    });

    // A key with no channel behind it settles only after the retrieve's not-found
    // grace period, so the name has to hold through both the wait and the failure.
    it("should mark a channel that is not found as missing", async () => {
      await renderInTaskFormWithClient(
        <Task.ChannelName channel={123456789} namePath={path.name} />,
        { client, values: { name: "form_name" } },
      );
      expect(screen.getByText("form_name")).toBeTruthy();
      await waitFor(() => expect(isMissing()).toBe(true), { timeout: 15000 });
      expect(await hoverTip("form_name")).toBe(
        "Channel not foundIt may have been deleted. Deploying the task creates a new " +
          "channel with this name.",
      );
    }, 20000);

    it("should rename the channel on the Core when an edit is committed", async () => {
      const ch = await createChannel();
      const editID = Task.getChannelNameID("live_ch");
      await renderInTaskFormWithClient(
        <Task.ChannelName channel={ch.key} namePath={path.name} id={editID} />,
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

    const createDeleted = async () => {
      const ch = await createChannel();
      await client.channels.delete(ch.key);
      return ch;
    };

    it("should write a rename of a missing channel to the form", async () => {
      const deleted = await createDeleted();
      const editID = Task.getChannelNameID("missing");
      const { form } = await renderInTaskFormWithClient(
        <Task.ChannelName channel={deleted.key} namePath={path.name} id={editID} />,
        { client, values: { name: "name_before" } },
      );
      await waitFor(() => expect(isMissing()).toBe(true));
      act(() => Text.edit(editID));
      const el = await awaitTextEditing(editID);
      act(() => commitTextEdit(el, "name_after"));
      await waitFor(() =>
        expect(form.current?.get(path.name).value).toBe("name_after"),
      );
    });

    it("should end the form name of a missing channel in the suffix", async () => {
      const deleted = await createDeleted();
      await renderInTaskFormWithClient(
        <Task.ChannelName channel={deleted.key} namePath={path.name} suffix="_int16" />,
        { client, values: { name: "form_name" } },
      );
      await waitFor(() => expect(isMissing()).toBe(true));
      expect(screen.getByText("form_name_int16")).toBeTruthy();
    });

    it("should say that deploying skips a missing skipped channel", async () => {
      const deleted = await createDeleted();
      await renderInTaskFormWithClient(
        <Task.ChannelName channel={deleted.key} namePath={path.name} skipped />,
        { client, values: { name: "form_name" } },
      );
      await waitFor(() => expect(isMissing()).toBe(true));
      expect(await hoverTip("form_name")).toBe(
        "Channel not foundIt may have been deleted. Deploying the task does not " +
          "create a disabled channel.",
      );
    });

    it("should ask for another channel when a picked one is missing", async () => {
      const deleted = await createDeleted();
      await renderInTaskFormWithClient(
        <Task.ChannelName channel={deleted.key} picked />,
        { client, values: { name: "" } },
      );
      await waitFor(() => expect(isMissing()).toBe(true));
      expect(await hoverTip("Channel not found")).toBe(
        "Channel not foundIt may have been deleted. Select another channel.",
      );
    });

    it("should allow no rename of a picked channel", async () => {
      const ch = await createChannel();
      await renderInTaskFormWithClient(<Task.ChannelName channel={ch.key} picked />, {
        client,
        values: { name: "" },
      });
      const name = await screen.findByText(ch.name);
      expect(name.className).not.toContain("lyra-text--editable");
    });

    describe("recovering from a failed lookup", () => {
      // Reads its key from the form so a spec can change it under a mounted name.
      const FormBound = () => {
        const channel = Form.useFieldValue(path.channel);
        return <Task.ChannelName channel={channel} namePath={path.name} />;
      };

      it("should show the channel once the row's key changes to one that resolves", async () => {
        const deleted = await createChannel();
        await client.channels.delete(deleted.key);
        const live = await createChannel();
        const { form } = await renderInTaskFormWithClient(<FormBound />, {
          client,
          values: { name: "form_name", channel: deleted.key },
        });
        await waitFor(() => expect(isMissing()).toBe(true));
        act(() => form.current?.set(path.channel, live.key));
        await screen.findByText(live.name);
        expect(isMissing()).toBe(false);
      });

      it("should show the channel again after a delete that fails", async () => {
        const index = await client.channels.create({
          name: uniqueName("idx"),
          dataType: DataType.TIMESTAMP,
          isIndex: true,
        });
        const writer = await client.openWriter({
          channels: [index.key],
          start: TimeStamp.now(),
        });
        try {
          await renderInTaskFormWithClient(
            <Task.ChannelName channel={index.key} namePath={path.name} />,
            { client, values: { name: "" } },
          );
          await screen.findByText(index.name);
          await expect(client.channels.delete(index.key)).rejects.toThrow(
            "unclosed writers",
          );
          await waitFor(() => expect(isMissing()).toBe(false));
        } finally {
          await writer.close();
        }
      });
    });
  });
});
