// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, type ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Explorer } from "@/feature/range/next/Explorer";
import { Modals } from "@/platform/modals";
import { uniqueRangeName } from "@/platform/range/testutil";
import {
  createConsoleWrapper,
  renderSuspended,
  uniqueName,
  uniqueToken,
} from "@/testutil";

const client = createTestClient();

const renderExplorer = async (): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  await renderSuspended(
    <>
      <Explorer />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  await screen.findByText("All ranges");
};

const createRange = async (
  name: string,
  extra: Partial<ranger.New> = {},
): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await client.ranges.create({
    name,
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(10))),
    ...extra,
  });
};

/** Creates a label, one range with it, and one range without it. */
const createLabeled = async (): Promise<{
  label: label.Label;
  labeled: ranger.Range;
  other: ranger.Range;
}> => {
  const label = await client.labels.create({
    name: uniqueName("label"),
    color: "#000000",
  });
  const labeled = await createRange(uniqueRangeName("labeled"), {
    labels: [label.key],
  });
  const other = await createRange(uniqueRangeName("other"));
  return { label, labeled, other };
};

const search = (term: string): void => {
  const input = screen.getByPlaceholderText<HTMLInputElement>("Search ranges...");
  fireEvent.change(input, { target: { value: term } });
};

/** @returns the list item of the named range. */
const itemOf = (name: string): HTMLElement => screen.getByRole("listitem", { name });

describe("range/next/Explorer", () => {
  describe("filter", () => {
    it("should keep the ranges with a picked label and restore the rest when its chip goes", async () => {
      const { label, labeled, other } = await createLabeled();
      await renderExplorer();
      await screen.findByText(other.name);
      fireEvent.click(await screen.findByRole("button", { name: "Filter" }));
      const labels = await screen.findByRole("button", {
        name: "Labels",
        expanded: false,
      });
      fireEvent.keyDown(labels, { key: "ArrowRight" });
      fireEvent.change(await screen.findByPlaceholderText("Search labels..."), {
        target: { value: label.name },
      });
      fireEvent.click(await screen.findByRole("option", { name: label.name }));
      await waitFor(() => {
        expect(screen.queryByText(other.name)).toBeNull();
        expect(screen.getByText(labeled.name)).toBeTruthy();
      });
      fireEvent.click(screen.getByRole("button", { name: "Remove Labels filter" }));
      await screen.findByText(other.name);
    });

    it("should show the chip of a stored view that filters by one label", async () => {
      const { label, labeled, other } = await createLabeled();
      // A 0.61 view stores a sole labels filter outside an and.
      const stored = await client.views.create({
        name: uniqueName("view"),
        type: "range",
        where: { labels: [label.key] },
      });
      await renderExplorer();
      fireEvent.click(await screen.findByRole("tab", { name: stored.name }));
      await screen.findByText(labeled.name);
      expect(screen.getByRole("button", { name: "Remove Labels filter" })).toBeTruthy();
      expect(screen.queryByText(other.name)).toBeNull();
    });
  });

  describe("search", () => {
    it("should show a match under its parent and leave out the other children", async () => {
      const token = uniqueToken();
      const parent = await createRange(uniqueRangeName("parent"));
      const match = await createRange(`${token} match`, { parent: parent.key });
      const sibling = await createRange(uniqueRangeName("sibling"), {
        parent: parent.key,
      });
      await renderExplorer();
      search(token);
      await screen.findByText(match.name);
      await waitFor(() =>
        within(itemOf(parent.name)).getByRole("button", { name: "Collapse" }),
      );
      expect(screen.queryByText(sibling.name)).toBeNull();
    });

    it("should show the first 100 matches and say so", async () => {
      const token = uniqueToken();
      const start = TimeStamp.now();
      await client.ranges.create(
        Array.from({ length: 101 }, (_, i) => ({
          name: `${token} ${i}`,
          timeRange: new TimeRange(start, start.add(TimeSpan.seconds(10))),
        })),
      );
      await renderExplorer();
      search(token);
      await screen.findByText(/Showing the first 100 matches\./);
    });
  });

  describe("stored view", () => {
    it("should save the sort and the layout to the view", async () => {
      const stored = await client.views.create({
        name: uniqueName("view"),
        type: "range",
      });
      await renderExplorer();
      fireEvent.click(await screen.findByRole("tab", { name: stored.name }));
      fireEvent.click(await screen.findByRole("button", { name: /Newest first/ }));
      fireEvent.click(await screen.findByText("Name"));
      fireEvent.click(screen.getByRole("button", { name: /List/ }));
      await waitFor(async () => {
        const saved = await client.views.retrieve({ key: stored.key });
        if (saved.type !== "range") throw new Error("not a range view");
        expect([saved.sort, saved.layout]).toEqual(["name", "list"]);
      });
    });
  });
});
