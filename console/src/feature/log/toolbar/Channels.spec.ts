// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, log, type Synnax as Client } from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { color, id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channels } from "@/feature/log/toolbar/Channels";
import { client, renderLog } from "@/feature/log/toolbar/testutil";

const createChannel = async (): Promise<channel.Key> => {
  const ch = await client.channels.create({
    name: id.create(),
    dataType: DataType.FLOAT32,
    virtual: true,
  });
  return ch.key;
};

const entry = (
  channelKey: channel.Key,
  overrides: Partial<log.ChannelEntry> = {},
): log.ChannelEntry =>
  log.channelEntryZ.parse({ channel: channelKey, color: color.ZERO, ...overrides });

const roles = new RoleClients(client);

const renderChannels = (channels: log.ChannelEntry[] = [], as?: Client) =>
  renderLog(Channels, { log: { channels }, as });

// The channel rows, not counting the add-channel row.
const channelRows = (): HTMLElement[] =>
  within(screen.getByRole("list", { name: "Log channels" })).queryAllByRole("listitem");

describe("log/toolbar/Channels", () => {
  it("renders no channel rows when there are no channels", async () => {
    await renderChannels();
    expect(await screen.findByText("Add channel")).toBeDefined();
    await waitFor(() => expect(channelRows()).toHaveLength(0));
  });

  it("renders a row for each channel", async () => {
    await renderChannels([entry(await createChannel()), entry(await createChannel())]);
    await waitFor(() => expect(channelRows()).toHaveLength(2));
  });

  it("renders the configured channel alias", async () => {
    await renderChannels([entry(await createChannel(), { alias: "pressure" })]);
    expect(await screen.findByDisplayValue("pressure")).toBeDefined();
  });

  it("renders an absent precision as Auto", async () => {
    await renderChannels([entry(await createChannel())]);
    const input = await screen.findByPlaceholderText("Auto");
    expect((input as HTMLInputElement).value).toEqual("");
  });

  it("clears the precision with the Auto button", async () => {
    const ch = await createChannel();
    const { key } = await renderChannels([entry(ch, { precision: 3 })]);
    expect(await screen.findByDisplayValue("3")).toBeDefined();
    fireEvent.click(await screen.findByLabelText("Enable auto precision"));
    await waitFor(async () =>
      expect(
        (await client.logs.retrieve({ key })).channels[0].precision,
      ).toBeUndefined(),
    );
  });

  it("removes a channel row with the clear button on its channel icon", async () => {
    const removed = await createChannel();
    const kept = await createChannel();
    await renderChannels([entry(removed), entry(kept)]);
    const { name } = await client.channels.retrieve(removed);
    fireEvent.click(await screen.findByLabelText(`Clear ${name}`));
    await waitFor(() => expect(channelRows()).toHaveLength(1));
    expect(screen.queryByLabelText(`Clear ${name}`)).toBeNull();
  });

  it("swaps a row's channel for the one picked in its select", async () => {
    const from = await createChannel();
    const to = await createChannel();
    await renderChannels([entry(from)]);
    const [fromCh, toCh] = await client.channels.retrieve([from, to]);
    await screen.findByLabelText(`Clear ${fromCh.name}`);
    fireEvent.click(screen.getAllByText(fromCh.name)[0]);
    const search = await screen.findByPlaceholderText("Search channels...");
    fireEvent.change(search, { target: { value: toCh.name } });
    fireEvent.click(await screen.findByRole("option", { name: new RegExp(toCh.name) }));
    expect(await screen.findByLabelText(`Clear ${toCh.name}`)).toBeDefined();
    expect(screen.queryByLabelText(`Clear ${fromCh.name}`)).toBeNull();
    expect(channelRows()).toHaveLength(1);
  });

  it("removes the row of a deleted channel with the clear button", async () => {
    const deleted = await createChannel();
    await client.channels.delete(deleted);
    await renderChannels([entry(deleted)]);
    await waitFor(() => expect(channelRows()).toHaveLength(1));
    fireEvent.click(await screen.findByLabelText("Clear selection"));
    await waitFor(() => expect(channelRows()).toHaveLength(0));
  });
});

describe("log/toolbar/Channels permissions", () => {
  const isAddRowFrozen = (result: { container: HTMLElement }) =>
    result.container.querySelector(".console-log__channel-select .lyra--disabled") !=
    null;

  it("should leave the add row live for a subject who may write the log", async () => {
    const ch = await createChannel();
    const { result } = await renderChannels([entry(ch)]);
    await waitFor(() => expect(channelRows()).toHaveLength(1));
    await waitFor(() => expect(isAddRowFrozen(result)).toBe(false));
  });

  it("should freeze the add row for a viewer", async () => {
    const ch = await createChannel();
    const { result } = await renderChannels([entry(ch)], await roles.get("Viewer"));
    await waitFor(() => expect(channelRows()).toHaveLength(1));
    await waitFor(() => expect(isAddRowFrozen(result)).toBe(true));
  });
});
