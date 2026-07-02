// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type ranger } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const TIMEOUT = { timeout: 5000 };

const createRange = async (): Promise<ranger.Range> => await createTestRange(client);

describe("Range.MetaData", () => {
  it("should render the metadata header", async () => {
    const range = await createRange();
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Range.MetaData rangeKey={range.key} />, { wrapper });
    await waitFor(() => expect(screen.getByText("Metadata")).toBeTruthy(), TIMEOUT);
  });

  it("should list the range's existing metadata pairs", async () => {
    const range = await createRange();
    const metaKey = id.create();
    const metaValue = id.create();
    await range.kv.set(metaKey, metaValue);
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Range.MetaData rangeKey={range.key} />, { wrapper });
    await waitFor(() => expect(screen.getByText(metaKey)).toBeTruthy(), TIMEOUT);
    expect(screen.getByDisplayValue(metaValue)).toBeTruthy();
  });

  it("should add a new metadata pair through the create form", async () => {
    const range = await createRange();
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Range.MetaData rangeKey={range.key} />, { wrapper });
    await waitFor(() => expect(screen.getByText("Metadata")).toBeTruthy(), TIMEOUT);

    const addButton = screen
      .getAllByRole("button")
      .find((b) => b.querySelector("svg[aria-label*='add']"));
    if (addButton == null) throw new Error("add button not found");
    fireEvent.click(addButton);

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
      .find((b) => b.querySelector("svg[aria-label*='check']"));
    if (saveButton == null) throw new Error("save button not found");
    fireEvent.click(saveButton);

    await waitFor(async () => {
      expect(await range.kv.get(metaKey)).toBe(metaValue);
    }, TIMEOUT);
  });
});
