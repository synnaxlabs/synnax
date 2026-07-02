// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type ranger } from "@synnaxlabs/client";
import { id, TimeRange } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Range } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const TIMEOUT = { timeout: 5000 };

const createRange = async (): Promise<ranger.Range> => await createTestRange(client);

const stubClipboard = (): ReturnType<typeof vi.fn> => {
  const writeText = vi.fn(async (_text: string) => {});
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText },
  });
  return writeText;
};

const buttonWithIcon = (label: string): HTMLElement => {
  const button = screen
    .getAllByRole("button")
    .find((b) => b.querySelector(`svg[aria-label*='${label}']`));
  if (button == null) throw new Error(`button with icon ${label} not found`);
  return button;
};

describe("Range.Details", () => {
  it("should load the range's name into the form", async () => {
    const range = await createRange();
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Range.Details rangeKey={range.key} />, { wrapper });
    await waitFor(
      () => expect(screen.getByDisplayValue(range.name)).toBeTruthy(),
      TIMEOUT,
    );
  });

  it("should render the from/to time range fields", async () => {
    const range = await createRange();
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Range.Details rangeKey={range.key} />, { wrapper });
    await waitFor(() => expect(screen.getByText("From")).toBeTruthy(), TIMEOUT);
    expect(screen.getByText("To")).toBeTruthy();
    expect(screen.getByText("Stage")).toBeTruthy();
  });

  it("should render the parent range button for a child range", async () => {
    const parent = await createRange();
    const child = await client.ranges.create({
      name: id.create(),
      timeRange: new TimeRange(1, 2),
      parent,
    });
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Range.Details rangeKey={child.key} />, { wrapper });
    await waitFor(
      () => expect(screen.getByText("Child Range of")).toBeTruthy(),
      TIMEOUT,
    );
    expect(screen.getByText(parent.name)).toBeTruthy();
  });

  it("should place the parent range's overview layout when its button is clicked", async () => {
    const parent = await createRange();
    const child = await client.ranges.create({
      name: id.create(),
      timeRange: new TimeRange(1, 2),
      parent,
    });
    const { wrapper, store } = await createConsoleWrapper({ client });
    render(<Range.Details rangeKey={child.key} />, { wrapper });
    fireEvent.click(await screen.findByText(parent.name, {}, TIMEOUT));
    await waitFor(
      () => expect(Session.Layout.select(store.getState(), parent.key)).toBeDefined(),
      TIMEOUT,
    );
  });

  it("should copy a link to the range to the clipboard", async () => {
    const writeText = stubClipboard();
    const range = await createRange();
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: {
        [Session.Cluster.SLICE_NAME]: {
          version: 0,
          selected: "local",
          clusters: {
            local: {
              key: "local",
              name: "Local",
              host: "localhost",
              port: 9090,
              username: "synnax",
              password: "seldon",
              secure: false,
            },
          },
        },
      },
    });
    render(<Range.Details rangeKey={range.key} />, { wrapper });
    await waitFor(
      () => expect(screen.getByDisplayValue(range.name)).toBeTruthy(),
      TIMEOUT,
    );
    fireEvent.click(buttonWithIcon("link"));
    await waitFor(() => expect(writeText).toHaveBeenCalled(), TIMEOUT);
    expect(writeText.mock.calls[0][0]).toContain(range.key);
  });

  it("should copy Python retrieval code to the clipboard", async () => {
    const writeText = stubClipboard();
    const range = await createRange();
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Range.Details rangeKey={range.key} />, { wrapper });
    await waitFor(
      () => expect(screen.getByDisplayValue(range.name)).toBeTruthy(),
      TIMEOUT,
    );
    fireEvent.click(buttonWithIcon("python"));
    await waitFor(() => expect(writeText).toHaveBeenCalled(), TIMEOUT);
    const copied = writeText.mock.calls[0][0];
    expect(copied).toContain("client.ranges.retrieve");
    expect(copied).toContain(range.key);
  });

  it("should copy TypeScript retrieval code to the clipboard", async () => {
    const writeText = stubClipboard();
    const range = await createRange();
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Range.Details rangeKey={range.key} />, { wrapper });
    await waitFor(
      () => expect(screen.getByDisplayValue(range.name)).toBeTruthy(),
      TIMEOUT,
    );
    fireEvent.click(buttonWithIcon("typescript"));
    await waitFor(() => expect(writeText).toHaveBeenCalled(), TIMEOUT);
    const copied = writeText.mock.calls[0][0];
    expect(copied).toContain("client.ranges.retrieve");
    expect(copied).toContain(range.key);
  });
});
