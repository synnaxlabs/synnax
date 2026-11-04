// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type log, type Synnax as Client } from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Properties } from "@/feature/log/toolbar/Properties";
import { client, renderLog } from "@/feature/log/toolbar/testutil";

const roles = new RoleClients(client);

const renderProperties = async (overrides: Partial<log.New> = {}, as?: Client) => {
  const { key, result } = await renderLog(Properties, { log: overrides, as });
  const checkboxes = () =>
    result.container.querySelectorAll<HTMLInputElement>('input[type="checkbox"]');
  // The switches render in document order: index 0 is Show Receipt Timestamp, the last
  // is Show Channel Names (Timestamp Precision is a numeric input, not a switch).
  return {
    key,
    showTimestamps: () => checkboxes()[0],
    showChannelNames: () => checkboxes()[checkboxes().length - 1],
  };
};

describe("log/toolbar/Properties", () => {
  it("renders the property controls", async () => {
    await renderProperties();
    expect(await screen.findByText("Show channel names")).toBeDefined();
    expect(screen.getByText("Timestamp precision")).toBeDefined();
    expect(screen.getByText("Show timestamps")).toBeDefined();
  });

  it("reflects showChannelNames from the server", async () => {
    const { showChannelNames } = await renderProperties({ channelNamesHidden: true });
    await waitFor(() => expect(showChannelNames().checked).toBe(false));
  });

  it("reflects showTimestamps from the server", async () => {
    const { showTimestamps } = await renderProperties({
      timestampsHidden: true,
    });
    await waitFor(() => expect(showTimestamps().checked).toBe(false));
  });

  it("toggles showChannelNames through the server", async () => {
    const { key, showChannelNames } = await renderProperties({
      channelNamesHidden: false,
    });
    await waitFor(() => expect(showChannelNames().disabled).toBe(false));
    fireEvent.click(showChannelNames());
    await waitFor(() => expect(showChannelNames().checked).toBe(false));
    await waitFor(async () =>
      expect((await client.logs.retrieve(key)).channelNamesHidden).toBe(true),
    );
  });

  it("toggles showTimestamps through the server", async () => {
    const { key, showTimestamps } = await renderProperties({
      timestampsHidden: false,
    });
    await waitFor(() => expect(showTimestamps().disabled).toBe(false));
    fireEvent.click(showTimestamps());
    await waitFor(() => expect(showTimestamps().checked).toBe(false));
    await waitFor(async () =>
      expect((await client.logs.retrieve(key)).timestampsHidden).toBe(true),
    );
  });
});

describe("log/toolbar/Properties permissions", () => {
  it("should freeze every control for a viewer", async () => {
    const { showChannelNames, showTimestamps } = await renderProperties(
      {},
      await roles.get("Viewer"),
    );
    await screen.findByText("Show channel names");
    await waitFor(() => expect(showChannelNames().disabled).toBe(true));
    expect(showTimestamps().disabled).toBe(true);
  });
});
