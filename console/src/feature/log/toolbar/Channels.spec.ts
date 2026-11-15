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

const createChannel = async (
  dataType: DataType = DataType.FLOAT32,
  name: string = id.create(),
): Promise<channel.Key> => {
  const ch = await client.channels.create({
    name,
    dataType,
    virtual: true,
  });
  return ch.key;
};

const entry = (
  channelKey: channel.Key,
  overrides: Partial<log.ChannelEntry> = {},
): log.ChannelEntry => log.channelEntryZ.parse({ channel: channelKey, ...overrides });

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

  it("limits an alias to 64 characters, counting an emoji as one", async () => {
    const key = await createChannel();
    const { key: logKey } = await renderChannels([entry(key)]);
    const { name } = await client.channels.retrieve(key);
    fireEvent.change(await screen.findByPlaceholderText(name), {
      target: { value: "😀".repeat(65) },
    });
    await waitFor(async () =>
      expect((await client.logs.retrieve(logKey)).channels[0].alias).toBe(
        "😀".repeat(64),
      ),
    );
  });

  it("renders an absent precision as an empty field with an auto chip", async () => {
    await renderChannels([entry(await createChannel())]);
    const [row] = await waitFor(() => {
      const rows = channelRows();
      expect(rows).toHaveLength(1);
      return rows;
    });
    expect(within(row).getByText("Auto")).toBeDefined();
    expect(within(row).queryByLabelText("Auto")).toBeNull();
  });

  it("clears the precision with the Auto button", async () => {
    const ch = await createChannel();
    const { key } = await renderChannels([entry(ch, { precision: 3 })]);
    expect(await screen.findByDisplayValue("3")).toBeDefined();
    fireEvent.click(await screen.findByLabelText("Auto"));
    await waitFor(async () =>
      expect(
        (await client.logs.retrieve({ key })).channels[0].precision,
      ).toBeUndefined(),
    );
  });

  it("rounds a typed fractional precision to a whole number", async () => {
    const ch = await createChannel();
    const { key } = await renderChannels([entry(ch, { precision: 3 })]);
    const input = await screen.findByDisplayValue("3");
    fireEvent.change(input, { target: { value: "4.6" } });
    fireEvent.blur(input);
    await waitFor(async () =>
      expect((await client.logs.retrieve({ key })).channels[0].precision).toBe(5),
    );
  });

  it("clamps a typed precision to 17", async () => {
    const ch = await createChannel();
    const { key } = await renderChannels([entry(ch, { precision: 3 })]);
    const input = await screen.findByDisplayValue("3");
    fireEvent.change(input, { target: { value: "40" } });
    fireEvent.blur(input);
    await waitFor(async () =>
      expect((await client.logs.retrieve({ key })).channels[0].precision).toBe(17),
    );
  });

  it("clears the precision when its field is emptied", async () => {
    const ch = await createChannel();
    const { key } = await renderChannels([entry(ch, { precision: 3 })]);
    const input = await screen.findByDisplayValue("3");
    fireEvent.change(input, { target: { value: "" } });
    fireEvent.blur(input);
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

  it("omits a channel already in the log from a row's select", async () => {
    const prefix = id.create();
    const row = await createChannel(DataType.FLOAT32, `${prefix}_row`);
    const other = await createChannel(DataType.FLOAT32, `${prefix}_other`);
    await createChannel(DataType.FLOAT32, `${prefix}_free`);
    await renderChannels([entry(row), entry(other)]);
    await screen.findByLabelText(`Clear ${prefix}_row`);
    fireEvent.click(screen.getAllByText(`${prefix}_row`)[0]);
    const search = await screen.findByPlaceholderText("Search channels...");
    fireEvent.change(search, { target: { value: prefix } });
    await screen.findByRole("option", { name: new RegExp(`${prefix}_free`) });
    expect(
      screen.queryByRole("option", { name: new RegExp(`${prefix}_other`) }),
    ).toBeNull();
  });

  it("offers a free channel past a page of channels in the log", async () => {
    const prefix = id.create();
    const used = await Promise.all(
      Array.from({ length: 11 }, (_, i) =>
        createChannel(DataType.FLOAT32, `${prefix}_${i}`),
      ),
    );
    // Search ranks a longer name lower, so this one misses the first page.
    const free = `${prefix}_free_channel_with_a_much_longer_name`;
    await createChannel(DataType.FLOAT32, free);
    await renderChannels(used.map((key) => entry(key)));
    await screen.findByLabelText(`Clear ${prefix}_0`);
    fireEvent.click(screen.getAllByText(`${prefix}_0`)[0]);
    const search = await screen.findByPlaceholderText("Search channels...");
    fireEvent.change(search, { target: { value: prefix } });
    await screen.findByRole("option", { name: new RegExp(free) });
  });

  it("offers a channel again once its row is removed", async () => {
    const prefix = id.create();
    const row = await createChannel(DataType.FLOAT32, `${prefix}_row`);
    const removed = await createChannel(DataType.FLOAT32, `${prefix}_removed`);
    await renderChannels([entry(row), entry(removed)]);
    fireEvent.click(await screen.findByLabelText(`Clear ${prefix}_removed`));
    await waitFor(() => expect(channelRows()).toHaveLength(1));
    fireEvent.click(screen.getAllByText(`${prefix}_row`)[0]);
    const search = await screen.findByPlaceholderText("Search channels...");
    fireEvent.change(search, { target: { value: prefix } });
    await screen.findByRole("option", { name: new RegExp(`${prefix}_removed`) });
  });

  it("keeps a row's open select and search when the log's channels change", async () => {
    const prefix = id.create();
    const row = await createChannel(DataType.FLOAT32, `${prefix}_row`);
    const other = await createChannel(DataType.FLOAT32, `${prefix}_other`);
    const { key } = await renderChannels([entry(row), entry(other)]);
    await screen.findByLabelText(`Clear ${prefix}_row`);
    fireEvent.click(screen.getAllByText(`${prefix}_row`)[0]);
    const search = await screen.findByPlaceholderText("Search channels...");
    fireEvent.change(search, { target: { value: prefix } });
    await screen.findByRole("option", { name: new RegExp(`${prefix}_row`) });
    const otherOption = { name: new RegExp(`${prefix}_other`) };
    expect(screen.queryByRole("option", otherOption)).toBeNull();

    await client.logs.dispatch(key, [log.removeChannel({ channel: other })]);

    await screen.findByRole("option", otherOption);
    expect(screen.getByPlaceholderText("Search channels...")).toBe(search);
    expect(search).toHaveProperty("value", prefix);
  });

  describe("color", () => {
    const swatchColor = async (): Promise<string> => {
      const [row] = await waitFor(() => {
        const rows = channelRows();
        expect(rows).toHaveLength(1);
        return rows;
      });
      const swatch = row.querySelector<HTMLElement>(".lyra-color-swatch");
      return swatch?.style.getPropertyValue("--pluto-swatch-color") ?? "";
    };

    it("shows the text color of the log when a channel has no color", async () => {
      await renderChannels([entry(await createChannel())]);
      const auto = await swatchColor();
      expect(auto).not.toBe("");
      expect(auto).not.toBe("#00000000");
    });

    it("shows a zero color as transparent black", async () => {
      await renderChannels([
        entry(await createChannel(), { color: color.construct("#00000000") }),
      ]);
      expect(await swatchColor()).toBe("#00000000");
    });
  });

  it("removes the row of a deleted channel", async () => {
    const deleted = await createChannel();
    await renderChannels([entry(deleted)]);
    await waitFor(() => expect(channelRows()).toHaveLength(1));
    // The Core removes the channel from the log, so its row goes.
    await client.channels.delete(deleted);
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

  const isFrozen = (element: Element | null | undefined): boolean =>
    element?.getAttribute("aria-disabled") === "true";

  const swatch = (result: { container: HTMLElement }): Element | null =>
    result.container.querySelector(".lyra-color-swatch");

  it("should leave a row's controls live for a subject who may write", async () => {
    const { result } = await renderChannels([entry(await createChannel())]);
    const standard = await screen.findByText("Standard");
    await waitFor(() => expect(isAddRowFrozen(result)).toBe(false));
    expect(isFrozen(standard.closest("button"))).toBe(false);
    expect(isFrozen(swatch(result))).toBe(false);
  });

  it("should freeze the number controls of a row for a viewer", async () => {
    const ch = await createChannel();
    const { result } = await renderChannels([entry(ch)], await roles.get("Viewer"));
    const standard = await screen.findByText("Standard");
    await waitFor(() => expect(isFrozen(standard.closest("button"))).toBe(true));
    expect(isFrozen(swatch(result))).toBe(true);
  });

  it("should freeze the timestamp controls of a row for a viewer", async () => {
    const ch = await createChannel(DataType.TIMESTAMP);
    await renderChannels([entry(ch)], await roles.get("Viewer"));
    const local = await screen.findByText("Local");
    await waitFor(() => expect(isFrozen(local.closest("button"))).toBe(true));
    expect(isFrozen(screen.getByText("Precise date").closest("button"))).toBe(true);
  });

  it("should freeze the add row for a viewer", async () => {
    const ch = await createChannel();
    const { result } = await renderChannels([entry(ch)], await roles.get("Viewer"));
    await waitFor(() => expect(channelRows()).toHaveLength(1));
    await waitFor(() => expect(isAddRowFrozen(result)).toBe(true));
  });
});
