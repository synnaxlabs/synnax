// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, log } from "@synnaxlabs/client";
import { color, id } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channels } from "@/layered/service/log/toolbar/Channels";
import { client, renderLog, TIMEOUT } from "@/layered/service/log/toolbar/testutil";

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

const renderChannels = (channels: log.ChannelEntry[] = []) =>
  renderLog(Channels, { log: { channels } });

const rowsIn = (result: { container: HTMLElement }) =>
  result.container.querySelectorAll(".console-log__channel-row");

describe("log/toolbar/Channels", () => {
  it("renders only the add-channel row when there are no channels", async () => {
    const { result } = await renderChannels();
    expect(await screen.findByText("Add a channel...")).toBeDefined();
    await waitFor(() => expect(rowsIn(result)).toHaveLength(1), TIMEOUT);
  });

  it("renders a row for each channel plus the add row", async () => {
    const { result } = await renderChannels([
      entry(await createChannel()),
      entry(await createChannel()),
    ]);
    await waitFor(() => expect(rowsIn(result)).toHaveLength(3), TIMEOUT);
  });

  it("renders the configured channel alias", async () => {
    await renderChannels([entry(await createChannel(), { alias: "pressure" })]);
    expect(await screen.findByDisplayValue("pressure")).toBeDefined();
  });

  it("renders an enabled remove button on each channel row for a permitted user", async () => {
    const { result } = await renderChannels([
      entry(await createChannel()),
      entry(await createChannel()),
    ]);
    await waitFor(() => {
      const removeButtons = result.container.querySelectorAll(
        ".console-log__channel-row button:not([disabled])",
      );
      expect(removeButtons.length).toBeGreaterThanOrEqual(2);
    }, TIMEOUT);
  });
});
