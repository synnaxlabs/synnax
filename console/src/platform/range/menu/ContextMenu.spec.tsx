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
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { color, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, onTestFinished } from "vitest";

import { Range } from "@/platform/range";
import {
  createPlannedTestRange,
  createTestRange,
  uniqueRangeName,
} from "@/platform/range/testutil";
import {
  createConsoleWrapper,
  openSubmenu,
  renderSuspended,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

/** Renders the menu of the ranges and waits for their items to load. */
const renderMenu = async (ranges: ranger.Range[]): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  await renderSuspended(<Range.ContextMenu keys={ranges.map((r) => r.key)} />, {
    wrapper,
  });
  await screen.findByRole("menuitem", { name: "Labels" });
};

const searchIn = (dialog: HTMLElement, placeholder: string, term: string): void => {
  fireEvent.change(within(dialog).getByPlaceholderText(placeholder), {
    target: { value: term },
  });
};

const labelsOf = async (rng: ranger.Range): Promise<label.Key[]> =>
  (await client.ranges.retrieve(rng.key)).labels;

const parentOf = async (rng: ranger.Range): Promise<ranger.Key | undefined> =>
  (await client.ranges.retrieve(rng.key)).parent;

describe("Range.ContextMenu", () => {
  it("should offer nothing but Reload Console with no ranges", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    await renderSuspended(<Range.ContextMenu keys={[]} />, { wrapper });
    await screen.findByRole("menuitem", { name: "Reload Console" });
    expect(screen.queryByRole("menuitem", { name: "Copy" })).toBeNull();
  });

  describe("labels", () => {
    it("should add a label that some ranges hold to all of them, then remove it from all", async () => {
      const lbl = await client.labels.create({
        name: uniqueName("label"),
        color: { type: "fixed", value: "#000000" },
      });
      const held = await createTestRange(client, {
        name: uniqueRangeName("held"),
        labels: [lbl.key],
      });
      const bare = await createTestRange(client, { name: uniqueRangeName("bare") });
      await renderMenu([held, bare]);
      const labels = await openSubmenu("Labels");
      searchIn(labels, "Search labels...", lbl.name);
      fireEvent.click(await within(labels).findByText(lbl.name));
      await waitFor(async () => {
        expect(await labelsOf(held)).toEqual([lbl.key]);
        expect(await labelsOf(bare)).toEqual([lbl.key]);
      });
      fireEvent.click(within(labels).getByText(lbl.name));
      await waitFor(async () => {
        expect(await labelsOf(held)).toEqual([]);
        expect(await labelsOf(bare)).toEqual([]);
      });
    });

    it("should create a label from the search text and add it to the ranges", async () => {
      const rng = await createTestRange(client, { name: uniqueRangeName("create") });
      await renderMenu([rng]);
      const labels = await openSubmenu("Labels");
      const name = uniqueName("created");
      searchIn(labels, "Search labels...", name);
      fireEvent.click(await within(labels).findByText(`Create label "${name}"`));
      const key = await waitFor(async () => {
        const [first] = await labelsOf(rng);
        if (first == null) throw new Error("no label on the range");
        return first;
      });
      expect((await client.labels.retrieve(key)).name).toBe(name);
    });
  });

  describe("parent", () => {
    it("should move the ranges under the picked range", async () => {
      const target = await createTestRange(client, { name: uniqueRangeName("target") });
      const ranges = [
        await createTestRange(client, { name: uniqueRangeName("moved") }),
        await createTestRange(client, { name: uniqueRangeName("moved") }),
      ];
      await renderMenu(ranges);
      const parent = await openSubmenu("Parent");
      searchIn(parent, "Search ranges...", target.name);
      fireEvent.click(await within(parent).findByText(target.name));
      await waitFor(async () => {
        for (const r of ranges) expect(await parentOf(r)).toBe(target.key);
      });
    });

    it("should move the ranges to the top level on No parent", async () => {
      const top = await createTestRange(client, { name: uniqueRangeName("top") });
      const child = await createTestRange(client, {
        name: uniqueRangeName("child"),
        parent: top.key,
      });
      await renderMenu([child]);
      fireEvent.click(within(await openSubmenu("Parent")).getByText("No parent"));
      await waitFor(async () => expect(await parentOf(child)).toBeUndefined());
    });

    it("should leave the ranges nested in the ranges out of the list", async () => {
      const token = uniqueName("nested");
      const sibling = await createTestRange(client, { name: `${token} sibling` });
      const self = await createTestRange(client, { name: `${token} self` });
      const child = await createTestRange(client, {
        name: `${token} child`,
        parent: self.key,
      });
      const grandchild = await createTestRange(client, {
        name: `${token} grandchild`,
        parent: child.key,
      });
      await renderMenu([self]);
      const parent = await openSubmenu("Parent");
      searchIn(parent, "Search ranges...", token);
      await within(parent).findByText(sibling.name);
      expect(within(parent).queryByText(child.name)).toBeNull();
      expect(within(parent).queryByText(grandchild.name)).toBeNull();
    });

    it("should list the other ranges when the nested ranges fill a page", async () => {
      // The ranges start after every other range, so they lead the newest-first list,
      // and the spec deletes them.
      const start = TimeStamp.now().add(TimeSpan.days(36500));
      const at = (offset: number) => {
        const from = start.add(TimeSpan.microseconds(offset));
        return { timeRange: new TimeRange(from, from.add(TimeSpan.microseconds(1))) };
      };
      const other = await createTestRange(client, {
        name: uniqueRangeName("other"),
        ...at(-1),
      });
      const self = await createTestRange(client, {
        name: uniqueRangeName("self"),
        ...at(0),
      });
      const nested = await client.ranges.create(
        Array.from({ length: 60 }, (_, i) => ({
          name: uniqueRangeName("nested"),
          parent: self.key,
          ...at(i + 1),
        })),
      );
      onTestFinished(
        async () =>
          await client.ranges.delete([
            other.key,
            self.key,
            ...nested.map((r) => r.key),
          ]),
      );
      await renderMenu([self]);
      const parent = await openSubmenu("Parent");
      expect(await within(parent).findByText(other.name)).toBeDefined();
    });

    it("should list the shared parent once, pinned at the top", async () => {
      const token = uniqueName("shared");
      const top = await createTestRange(client, { name: `${token} top` });
      const other = await createTestRange(client, { name: `${token} other` });
      const child = await createTestRange(client, {
        name: `${token} child`,
        parent: top.key,
      });
      await renderMenu([child]);
      const parent = await openSubmenu("Parent");
      searchIn(parent, "Search ranges...", token);
      await within(parent).findByText(other.name);
      expect(within(parent).getAllByText(top.name)).toHaveLength(1);
    });

    it("should leave the ranges themselves out of the list", async () => {
      const token = uniqueName("self");
      const sibling = await createTestRange(client, { name: `${token} sibling` });
      const self = await createTestRange(client, { name: `${token} self` });
      await renderMenu([self]);
      const parent = await openSubmenu("Parent");
      searchIn(parent, "Search ranges...", token);
      await within(parent).findByText(sibling.name);
      expect(within(parent).queryByText(self.name)).toBeNull();
    });
  });

  describe("color", () => {
    it("should set the color of every range when the submenu closes", async () => {
      const ranges = [
        await createTestRange(client, { name: uniqueRangeName("colored") }),
        await createTestRange(client, { name: uniqueRangeName("colored") }),
      ];
      await renderMenu(ranges);
      const dialog = await openSubmenu("Color");
      fireEvent.click(await within(dialog).findByText("Custom"));
      fireEvent.change(within(dialog).getByLabelText("Hex"), {
        target: { value: "0000ff" },
      });
      fireEvent.keyDown(dialog, { key: "ArrowLeft" });
      await waitFor(() =>
        expect(screen.queryByRole("dialog", { name: "Color" })).toBeNull(),
      );
      await waitFor(async () => {
        for (const r of ranges) {
          const { color: stored } = await client.ranges.retrieve(r.key);
          expect(stored).toEqual({ type: "fixed", value: color.construct("#0000ff") });
        }
      });
    });

    it("should link every range to a named color picked from a palette", async () => {
      const palette = await client.groups.create({
        scope: "color",
        name: uniqueName("palette"),
      });
      const named = await client.colors.create({
        name: uniqueName("color"),
        light: "#00ff00",
        group: palette.key,
      });
      const ranges = [
        await createTestRange(client, { name: uniqueRangeName("linked") }),
        await createTestRange(client, { name: uniqueRangeName("linked") }),
      ];
      await renderMenu(ranges);
      const dialog = await openSubmenu("Color");
      const swatch = await within(dialog).findByRole("button", { name: named.name });
      fireEvent.click(swatch);
      fireEvent.keyDown(swatch, { key: "ArrowLeft" });
      await waitFor(() =>
        expect(screen.queryByRole("dialog", { name: "Color" })).toBeNull(),
      );
      await waitFor(async () => {
        for (const r of ranges) {
          const { color: stored } = await client.ranges.retrieve(r.key);
          expect(stored).toEqual({ type: "named", key: named.key });
        }
      });
    });
  });

  describe("copy", () => {
    it("should offer no time range for a range planned with no times", async () => {
      const planned = await createPlannedTestRange(client, uniqueRangeName("planned"));
      await renderMenu([planned]);
      const copy = await openSubmenu("Copy");
      within(copy).getByRole("menuitem", { name: "Name" });
      expect(within(copy).queryByRole("menuitem", { name: "Time range" })).toBeNull();
    });

    it("should copy the names of every range, one per line", async () => {
      const writeText = stubClipboardWriteText();
      const ranges = [
        await createTestRange(client, { name: uniqueRangeName("copied") }),
        await createTestRange(client, { name: uniqueRangeName("copied") }),
      ];
      await renderMenu(ranges);
      const copy = await openSubmenu("Copy");
      fireEvent.click(within(copy).getByRole("menuitem", { name: "Names" }));
      await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
      expect(writeText.mock.calls[0][0].split("\n").sort()).toEqual(
        ranges.map((r) => r.name).sort(),
      );
    });
  });
});
