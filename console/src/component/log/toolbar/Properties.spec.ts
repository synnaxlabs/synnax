// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type log } from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Properties } from "@/service/log/toolbar/Properties";
import { client, renderLog, TIMEOUT } from "@/service/log/toolbar/testutil";

const renderProperties = async (overrides: Partial<log.New> = {}) => {
  const { key, result } = await renderLog(Properties, { log: overrides });
  const checkboxes = () =>
    result.container.querySelectorAll<HTMLInputElement>('input[type="checkbox"]');
  // The switches render in document order: index 0 is Show Receipt Timestamp, the last
  // is Show Channel Names (Receipt Timestamp Precision is a numeric input, not a switch).
  return {
    key,
    showReceiptTimestamp: () => checkboxes()[0],
    showChannelNames: () => checkboxes()[checkboxes().length - 1],
  };
};

describe("log/toolbar/Properties", () => {
  it("renders the property controls", async () => {
    await renderProperties();
    expect(await screen.findByText("Show Channel Names")).toBeDefined();
    expect(screen.getByText("Receipt Timestamp Precision")).toBeDefined();
    expect(screen.getByText("Show Receipt Timestamp")).toBeDefined();
  });

  it("reflects showChannelNames from the server", async () => {
    const { showChannelNames } = await renderProperties({ hideChannelNames: true });
    await waitFor(() => expect(showChannelNames().checked).toBe(false), TIMEOUT);
  });

  it("reflects showReceiptTimestamp from the server", async () => {
    const { showReceiptTimestamp } = await renderProperties({
      hideReceiptTimestamp: true,
    });
    await waitFor(() => expect(showReceiptTimestamp().checked).toBe(false), TIMEOUT);
  });

  it("toggles showChannelNames through the server", async () => {
    const { key, showChannelNames } = await renderProperties({
      hideChannelNames: false,
    });
    await waitFor(() => expect(showChannelNames().disabled).toBe(false), TIMEOUT);
    fireEvent.click(showChannelNames());
    await waitFor(() => expect(showChannelNames().checked).toBe(false), TIMEOUT);
    await waitFor(
      async () =>
        expect((await client.logs.retrieve({ key })).hideChannelNames).toBe(true),
      TIMEOUT,
    );
  });

  it("toggles showReceiptTimestamp through the server", async () => {
    const { key, showReceiptTimestamp } = await renderProperties({
      hideReceiptTimestamp: false,
    });
    await waitFor(() => expect(showReceiptTimestamp().disabled).toBe(false), TIMEOUT);
    fireEvent.click(showReceiptTimestamp());
    await waitFor(() => expect(showReceiptTimestamp().checked).toBe(false), TIMEOUT);
    await waitFor(
      async () =>
        expect((await client.logs.retrieve({ key })).hideReceiptTimestamp).toBe(true),
      TIMEOUT,
    );
  });
});
