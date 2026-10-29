// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const createRange = async (): Promise<ranger.Range> => await createTestRange(client);

const renderMetaData = async (rangeKey: string): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Range.MetaData rangeKey={rangeKey} />, { wrapper });
};

const getAddButton = (): HTMLElement => {
  const button = screen
    .getAllByRole("button")
    .find((b) => b.querySelector("svg.lyra-icon--add"));
  if (button == null) throw new Error("add button not found");
  return button;
};

describe("Range.MetaData", () => {
  it("should list the range's existing metadata pairs", async () => {
    const range = await createRange();
    const metaKey = id.create();
    const metaValue = id.create();
    await range.kv.set(metaKey, metaValue);
    await renderMetaData(range.key);
    await waitFor(() => expect(screen.getByText(metaKey)).toBeTruthy());
    expect(screen.getByDisplayValue(metaValue)).toBeTruthy();
  });

  it("should add a new metadata pair through the create form", async () => {
    const range = await createRange();
    await renderMetaData(range.key);
    await waitFor(() => expect(screen.getByText("Metadata")).toBeTruthy());
    fireEvent.click(getAddButton());

    const metaKey = id.create();
    const metaValue = id.create();
    fireEvent.change(screen.getByPlaceholderText("Key"), {
      target: { value: metaKey },
    });
    fireEvent.change(screen.getByPlaceholderText("Value"), {
      target: { value: metaValue },
    });

    const saveButton = screen
      .getAllByRole("button")
      .find((b) => b.querySelector("svg.lyra-icon--check"));
    if (saveButton == null) throw new Error("save button not found");
    fireEvent.click(saveButton);

    await waitFor(async () => {
      expect(await range.kv.get(metaKey)).toBe(metaValue);
    });
  });

  it("should focus the create form's key only once the form shows", async () => {
    const range = await createRange();
    await renderMetaData(range.key);
    const key = await screen.findByPlaceholderText("Key");
    expect(document.activeElement).not.toBe(key);
    fireEvent.click(getAddButton());
    await waitFor(() => expect(document.activeElement).toBe(key));
  });

  it("should store an edit to a listed value", async () => {
    const range = await createRange();
    const metaKey = id.create();
    await range.kv.set(metaKey, "before");
    await renderMetaData(range.key);
    const input = await screen.findByDisplayValue("before");
    fireEvent.change(input, { target: { value: "after" } });
    fireEvent.blur(input);
    await waitFor(async () => expect(await range.kv.get(metaKey)).toBe("after"));
  });

  it("should show a value that another client sets", async () => {
    const range = await createRange();
    const metaKey = id.create();
    await range.kv.set(metaKey, "before");
    await renderMetaData(range.key);
    await screen.findByDisplayValue("before");
    await waitForStreamLive(client.connection);
    await act(async () => {
      await createTestClient().ranges.getKV(range.key).set(metaKey, "remote");
    });
    expect(await screen.findByDisplayValue("remote")).toBeTruthy();
  });

  it("should remove a pair that another client deletes", async () => {
    const range = await createRange();
    const metaKey = id.create();
    await range.kv.set(metaKey, "doomed");
    await renderMetaData(range.key);
    await screen.findByText(metaKey);
    await waitForStreamLive(client.connection);
    await act(async () => {
      await createTestClient().ranges.getKV(range.key).delete(metaKey);
    });
    await waitFor(() => expect(screen.queryByText(metaKey)).toBeNull());
  });
});
