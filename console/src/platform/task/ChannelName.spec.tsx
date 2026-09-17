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

const unresolved = { device: undefined, resolve: () => 0, channelPath: "channel" };

describe("ChannelName", () => {
  it("should render the default name when no channel is selected", async () => {
    await renderInTaskForm(
      <Task.ChannelName
        channel={0}
        namePath="config.name"
        defaultName="No channel"
        {...unresolved}
      />,
      { values: { config: { name: "" } } },
    );
    await waitFor(() => expect(screen.getByText("No channel")).toBeTruthy());
  });

  it("should prefer the form name over the default when set", async () => {
    await renderInTaskForm(
      <Task.ChannelName
        channel={0}
        namePath="config.name"
        defaultName="No channel"
        {...unresolved}
      />,
      { values: { config: { name: "Manually Named" } } },
    );
    await waitFor(() => expect(screen.getByText("Manually Named")).toBeTruthy());
  });

  it("should write a committed rename to the form when no channel is bound", async () => {
    const editID = Task.getChannelNameID("unbound");
    const { form } = await renderInTaskForm(
      <Task.ChannelName
        channel={0}
        namePath="config.name"
        id={editID}
        {...unresolved}
      />,
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
        <Task.ChannelName channel={ch.key} namePath="name" {...unresolved} />,
        { client, values: { name: "" } },
      );
      await waitFor(() => expect(screen.getByText(ch.name)).toBeTruthy());
    });

    // A key with no channel behind it settles only after the retrieve's not-found
    // grace period, so the name has to hold through both the wait and the failure.
    it("should keep showing the form name for a channel that never resolves", async () => {
      await renderInTaskFormWithClient(
        <Task.ChannelName channel={123456789} namePath="name" {...unresolved} />,
        { client, values: { name: "form_name" } },
      );
      expect(screen.getByText("form_name")).toBeTruthy();
      await waitFor(
        () =>
          expect(
            screen.getByText("form_name").closest(".pluto--status-error"),
          ).toBeTruthy(),
        { timeout: 15000 },
      );
    }, 20000);

    it("should rename the channel on the Core when an edit is committed", async () => {
      const ch = await createChannel();
      const editID = Task.getChannelNameID("live_ch");
      await renderInTaskFormWithClient(
        <Task.ChannelName
          channel={ch.key}
          namePath="name"
          id={editID}
          {...unresolved}
        />,
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

    describe("resolving from the device map", () => {
      it("should keep the row's channel while the device is undefined", async () => {
        const ch = await createChannel();
        const other = await createChannel();
        const { form } = await renderInTaskFormWithClient(
          <Task.ChannelName
            channel={ch.key}
            channelPath="channel"
            namePath="name"
            device={undefined}
            resolve={() => other.key}
          />,
          { client, values: { name: "", channel: ch.key } },
        );
        await screen.findByText(ch.name);
        expect(screen.queryByText(other.name)).toBeNull();
        expect(form.current?.get("channel").value).toBe(ch.key);
      });

      it("should show the channel the resolver picks over the row's own", async () => {
        const stale = await createChannel();
        const bound = await createChannel();
        await renderInTaskFormWithClient(
          <Task.ChannelName
            channel={stale.key}
            channelPath="channel"
            namePath="name"
            device={{}}
            resolve={() => bound.key}
          />,
          { client, values: { name: "", channel: stale.key } },
        );
        await screen.findByText(bound.name);
        expect(screen.queryByText(stale.name)).toBeNull();
      });

      it("should show no channel when the resolver finds none, even if the row holds one", async () => {
        const stale = await createChannel();
        await renderInTaskFormWithClient(
          <Task.ChannelName
            channel={stale.key}
            channelPath="channel"
            namePath="name"
            defaultName="No channel"
            device={{}}
            resolve={() => 0}
          />,
          { client, values: { name: "", channel: stale.key } },
        );
        await screen.findByText("No channel");
        expect(screen.queryByText(stale.name)).toBeNull();
      });

      it("should hand the resolver the device it was given", async () => {
        const bound = await createChannel();
        const dev = { map: { port: bound.key } };
        await renderInTaskFormWithClient(
          <Task.ChannelName
            channel={0}
            channelPath="channel"
            namePath="name"
            device={dev}
            resolve={({ map }) => map.port}
          />,
          { client, values: { name: "", channel: 0 } },
        );
        await screen.findByText(bound.name);
      });

      it("should bind the row to the resolved channel in the form", async () => {
        const bound = await createChannel();
        const { form } = await renderInTaskFormWithClient(
          <Task.ChannelName
            channel={0}
            channelPath="channel"
            namePath="name"
            device={{}}
            resolve={() => bound.key}
          />,
          { client, values: { name: "", channel: 0 } },
        );
        await waitFor(() => expect(form.current?.get("channel").value).toBe(bound.key));
      });

      it("should rebind the row when the resolver picks a different channel", async () => {
        const stale = await createChannel();
        const bound = await createChannel();
        const { form } = await renderInTaskFormWithClient(
          <Task.ChannelName
            channel={stale.key}
            channelPath="channel"
            namePath="name"
            device={{}}
            resolve={() => bound.key}
          />,
          { client, values: { name: "", channel: stale.key } },
        );
        await waitFor(() => expect(form.current?.get("channel").value).toBe(bound.key));
      });

      it("should keep the row's channel when the resolver finds none", async () => {
        const stale = await createChannel();
        const { form } = await renderInTaskFormWithClient(
          <Task.ChannelName
            channel={stale.key}
            channelPath="channel"
            namePath="name"
            defaultName="No channel"
            device={{}}
            resolve={() => 0}
          />,
          { client, values: { name: "", channel: stale.key } },
        );
        await screen.findByText("No channel");
        await act(async () => {});
        expect(form.current?.get("channel").value).toBe(stale.key);
      });

      it("should not bind the row in preview mode", async () => {
        const bound = await createChannel();
        const { form } = await renderInTaskFormWithClient(
          <Task.ChannelName
            channel={0}
            channelPath="channel"
            namePath="name"
            device={{}}
            resolve={() => bound.key}
          />,
          { client, mode: "preview", values: { name: "", channel: 0 } },
        );
        await screen.findByText(bound.name);
        await act(async () => {});
        expect(form.current?.get("channel").value).toBe(0);
      });

      it("should rename the resolved channel rather than the row's form name", async () => {
        const bound = await createChannel();
        const editID = Task.getChannelNameID("resolved_ch");
        const { form } = await renderInTaskFormWithClient(
          <Task.ChannelName
            channel={0}
            channelPath="channel"
            namePath="name"
            id={editID}
            device={{}}
            resolve={() => bound.key}
          />,
          { client, values: { name: "form_name", channel: 0 } },
        );
        await screen.findByText(bound.name);
        Text.edit(editID);
        const el = await awaitTextEditing(editID);
        const newName = uniqueName("renamed");
        act(() => commitTextEdit(el, newName));
        await waitFor(async () => {
          const renamed = await client.channels.retrieve(bound.key);
          expect(renamed.name).toBe(newName);
        });
        expect(form.current?.get("name").value).toBe("form_name");
      });
    });
  });
});
