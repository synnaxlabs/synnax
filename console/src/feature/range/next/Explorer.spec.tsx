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
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

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

const search = async (term: string): Promise<void> => {
  const input =
    await screen.findByPlaceholderText<HTMLInputElement>("Search ranges...");
  fireEvent.change(input, { target: { value: term } });
};

/** @returns the list item of the named range. */
const itemOf = (name: string): HTMLElement => screen.getByRole("listitem", { name });

/** Waits for the list item of the named range. A search tints its term in names. */
const findItem = async (name: string): Promise<HTMLElement> =>
  await screen.findByRole("listitem", { name });

const SWAP = "console-range-filter__picker--swap";

/** jsdom lays nothing out, so a spec places the values beside the menu by hand. */
const BESIDE: DOMRect = {
  left: 150,
  top: 0,
  width: 200,
  height: 200,
  right: 350,
  bottom: 200,
  x: 150,
  y: 0,
  toJSON: () => "",
};

/** Opens the filter menu, then holds the clock so its timers run on demand. */
const openFilterMenu = async (): Promise<void> => {
  await renderExplorer();
  fireEvent.click(await screen.findByRole("button", { name: "Filter" }));
  await screen.findByRole("button", { name: "Labels", expanded: false });
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "performance"] });
};

/** @returns the closed property of the filter menu with the name. */
const closed = (name: string): HTMLElement =>
  screen.getByRole("button", { name, expanded: false });

/** @returns the open values of the property with the name. */
const values = (name: string): HTMLElement => screen.getByRole("dialog", { name });

const move = (target: HTMLElement, clientX: number, clientY: number): void => {
  fireEvent.pointerMove(target, { pointerType: "mouse", clientX, clientY });
};

const advance = (ms: number): void => {
  act(() => {
    vi.advanceTimersByTime(ms);
  });
};

/** Opens the labels, then moves the pointer across stage toward their values. */
const aimAtLabels = async (): Promise<void> => {
  await openFilterMenu();
  fireEvent.keyDown(closed("Labels"), { key: "ArrowRight" });
  vi.spyOn(values("Labels"), "getBoundingClientRect").mockReturnValue(BESIDE);
  move(screen.getByRole("button", { name: "Labels", expanded: true }), 100, 14);
  move(closed("Stage"), 140, 32);
};

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

  describe("filter menu", () => {
    afterEach(() => {
      act(() => {
        vi.runOnlyPendingTimers();
      });
      vi.useRealTimers();
    });

    it("should open the values of a property once the pointer rests on it", async () => {
      await openFilterMenu();
      move(closed("Time"), 50, 70);
      advance(100);
      expect(values("Time").className).not.toContain(SWAP);
    });

    it("should not open the values of a property the pointer only passes", async () => {
      await openFilterMenu();
      move(closed("Stage"), 50, 42);
      advance(60);
      move(closed("Time"), 50, 70);
      advance(100);
      // Values that opened over the values of stage would carry the swap.
      expect(values("Time").className).not.toContain(SWAP);
      expect(screen.queryByRole("dialog", { name: "Stage" })).toBeNull();
    });

    it("should open the values on ArrowRight and close them on ArrowLeft", async () => {
      await openFilterMenu();
      const labels = closed("Labels");
      fireEvent.keyDown(labels, { key: "ArrowRight" });
      fireEvent.keyDown(screen.getByPlaceholderText("Search labels..."), {
        key: "ArrowLeft",
      });
      expect(screen.queryByRole("dialog", { name: "Labels" })).toBeNull();
      expect(document.activeElement).toBe(labels);
    });

    it("should keep the open values while the pointer crosses another property toward them", async () => {
      await aimAtLabels();
      advance(250);
      move(screen.getByPlaceholderText("Search labels..."), 200, 50);
      advance(300);
      expect(values("Labels")).toBeTruthy();
      expect(screen.queryByRole("dialog", { name: "Stage" })).toBeNull();
    });

    it("should swap in the values under the pointer once the aim lapses", async () => {
      await aimAtLabels();
      advance(300);
      expect(values("Stage").className).toContain(SWAP);
      expect(screen.queryByRole("dialog", { name: "Labels" })).toBeNull();
    });

    it("should end the aim when the pointer heads past the values", async () => {
      await aimAtLabels();
      move(closed("Stage"), 145, 180);
      advance(100);
      expect(values("Stage").className).toContain(SWAP);
    });

    it("should end the aim when the pointer drops without moving toward the values", async () => {
      await aimAtLabels();
      move(closed("Stage"), 141, 50);
      advance(100);
      expect(values("Stage").className).toContain(SWAP);
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
      await search(token);
      await findItem(match.name);
      // The list reads no page for the tree until it reads the term.
      within(itemOf(parent.name)).getByRole("button", { name: "Collapse" });
      expect(screen.queryByText(sibling.name)).toBeNull();
    });

    it("should drop a deleted range that holds matches", async () => {
      const token = uniqueToken();
      const holder = await createRange(uniqueRangeName("holder"));
      const other = await createRange(uniqueRangeName("other"));
      const gone = await createRange(`${token} gone`, { parent: holder.key });
      const kept = await createRange(`${token} kept`, { parent: other.key });
      await renderExplorer();
      await search(token);
      await findItem(gone.name);
      within(itemOf(holder.name)).getByRole("button", { name: "Collapse" });
      await findItem(kept.name);
      await client.ranges.delete([gone.key, holder.key]);
      await waitFor(() =>
        expect(screen.queryByRole("listitem", { name: holder.name })).toBeNull(),
      );
      expect(itemOf(kept.name)).toBeTruthy();
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
      await search(token);
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
