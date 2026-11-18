// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, ranger, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { color, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Modals } from "@/platform/modals";
import { Range } from "@/platform/range";
import { uniqueRangeName } from "@/platform/range/testutil";
import {
  createConsoleWrapper,
  createTestClientWithGrants,
  openSubmenu,
  renderSuspended,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

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

/** Renders the menu of the ranges and waits for their items to load. */
const renderMenu = async (
  ranges: ranger.Range[],
  as: Client = client,
): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  await renderSuspended(
    <>
      <Range.ContextMenu keys={ranges.map((r) => r.key)} />
      <Modals.Stack />
    </>,
    { wrapper },
  );
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

/** Two ranges under one parent, at different times. */
const createSiblings = async (): Promise<{
  top: ranger.Range;
  ranges: ranger.Range[];
}> => {
  const top = await createRange(uniqueRangeName("top"));
  const start = TimeStamp.now();
  const ranges = await Promise.all(
    [0, 30].map(
      async (offset) =>
        await client.ranges.create({
          name: uniqueRangeName("moved"),
          timeRange: new TimeRange(
            start.add(TimeSpan.seconds(offset)),
            start.add(TimeSpan.seconds(offset + 10)),
          ),
          parent: top.key,
        }),
    ),
  );
  return { top, ranges };
};

const spanOf = (ranges: ranger.Range[]): TimeRange =>
  TimeRange.max(...ranges.map((r) => r.timeRange));

/** Waits for the Core to hold a range with the name, and returns it. */
const findCreated = async (name: string): Promise<ranger.Range> =>
  await waitFor(async () => await client.ranges.retrieve(name));

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
        color: "#000000",
      });
      const held = await createRange(uniqueRangeName("held"), { labels: [lbl.key] });
      const bare = await createRange(uniqueRangeName("bare"));
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
      const rng = await createRange(uniqueRangeName("create"));
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
      const target = await createRange(uniqueRangeName("target"));
      const ranges = [
        await createRange(uniqueRangeName("moved")),
        await createRange(uniqueRangeName("moved")),
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
      const top = await createRange(uniqueRangeName("top"));
      const child = await createRange(uniqueRangeName("child"), { parent: top.key });
      await renderMenu([child]);
      fireEvent.click(within(await openSubmenu("Parent")).getByText("No parent"));
      await waitFor(async () => expect(await parentOf(child)).toBeUndefined());
    });

    it("should list the shared parent once, pinned at the top", async () => {
      const token = uniqueName("shared");
      const top = await createRange(`${token} top`);
      const other = await createRange(`${token} other`);
      const child = await createRange(`${token} child`, { parent: top.key });
      await renderMenu([child]);
      const parent = await openSubmenu("Parent");
      searchIn(parent, "Search ranges...", token);
      await within(parent).findByText(other.name);
      expect(within(parent).getAllByText(top.name)).toHaveLength(1);
    });

    it("should leave the ranges themselves out of the list", async () => {
      const token = uniqueName("self");
      const sibling = await createRange(`${token} sibling`);
      const self = await createRange(`${token} self`);
      await renderMenu([self]);
      const parent = await openSubmenu("Parent");
      searchIn(parent, "Search ranges...", token);
      await within(parent).findByText(sibling.name);
      expect(within(parent).queryByText(self.name)).toBeNull();
    });

    it("should list the current parent once", async () => {
      const token = uniqueName("current");
      const top = await createRange(`${token} top`);
      const sibling = await createRange(`${token} sibling`);
      const child = await createRange(uniqueRangeName("child"), { parent: top.key });
      await renderMenu([child]);
      const parent = await openSubmenu("Parent");
      searchIn(parent, "Search ranges...", token);
      await within(parent).findByText(sibling.name);
      expect(within(parent).getAllByText(top.name)).toHaveLength(1);
    });

    it("should hide Create range while the search matches the current parent", async () => {
      const top = await createRange(uniqueRangeName("top"));
      const child = await createRange(uniqueRangeName("child"), { parent: top.key });
      await renderMenu([child]);
      const parent = await openSubmenu("Parent");
      searchIn(parent, "Search ranges...", top.name);
      expect(within(parent).getByText(top.name)).toBeTruthy();
      expect(within(parent).queryByText(/^Create range/)).toBeNull();
    });

    it("should list New range above No parent under the divider", async () => {
      await renderMenu([await createRange(uniqueRangeName("order"))]);
      const options = within(await openSubmenu("Parent")).getAllByRole("option");
      expect(options.slice(-2).map((o) => o.textContent)).toEqual([
        "New range",
        "No parent",
      ]);
    });

    it("should create a range from the search text that covers the ranges and holds them", async () => {
      const { top, ranges } = await createSiblings();
      await renderMenu(ranges);
      const parent = await openSubmenu("Parent");
      const name = uniqueRangeName("group");
      searchIn(parent, "Search ranges...", name);
      fireEvent.click(await within(parent).findByText(`Create range "${name}"`));
      const created = await findCreated(name);
      await waitFor(async () => {
        for (const r of ranges) expect(await parentOf(r)).toBe(created.key);
      });
      expect(created.timeRange.equals(spanOf(ranges))).toBe(true);
      expect(await parentOf(created)).toBe(top.key);
    });

    it("should hide Create range while the search matches a range name in any case", async () => {
      const target = await createRange(uniqueRangeName("named"));
      await renderMenu([await createRange(uniqueRangeName("moved"))]);
      const parent = await openSubmenu("Parent");
      searchIn(parent, "Search ranges...", target.name.toUpperCase());
      await within(parent).findByText(target.name);
      expect(within(parent).queryByText(/^Create range/)).toBeNull();
    });

    it("should open New range over the ranges and move them into the range it creates", async () => {
      const { top, ranges } = await createSiblings();
      await renderMenu(ranges);
      fireEvent.click(within(await openSubmenu("Parent")).getByText("New range"));
      const name = uniqueRangeName("group");
      fireEvent.change(await screen.findByPlaceholderText("Range name"), {
        target: { value: name },
      });
      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: "Create" }));
      });
      const created = await findCreated(name);
      await waitFor(async () => {
        for (const r of ranges) expect(await parentOf(r)).toBe(created.key);
      });
      // The form holds times as floats, which round off nanoseconds.
      expect(created.timeRange.equals(spanOf(ranges), TimeSpan.MICROSECOND)).toBe(true);
      expect(await parentOf(created)).toBe(top.key);
    });

    it("should withhold New range from a subject who cannot create ranges", async () => {
      const rng = await createRange(uniqueRangeName("guarded"));
      const as = await createTestClientWithGrants(client, {
        retrieve: [ranger.TYPE_RESOURCE_ID],
        update: [ranger.TYPE_RESOURCE_ID],
      });
      await renderMenu([rng], as);
      const parent = await openSubmenu("Parent");
      within(parent).getByText("No parent");
      expect(within(parent).queryByText("New range")).toBeNull();
    });
  });

  describe("color", () => {
    it("should set the color of every range when the submenu closes", async () => {
      const ranges = [
        await createRange(uniqueRangeName("colored")),
        await createRange(uniqueRangeName("colored")),
      ];
      await renderMenu(ranges);
      const dialog = await openSubmenu("Color");
      const swatch = within(dialog).getAllByLabelText(/^#/)[0];
      const hex = swatch.getAttribute("aria-label");
      fireEvent.click(swatch);
      fireEvent.keyDown(swatch, { key: "ArrowLeft" });
      await waitFor(() =>
        expect(screen.queryByRole("dialog", { name: "Color" })).toBeNull(),
      );
      await waitFor(async () => {
        for (const r of ranges) {
          const { color: stored } = await client.ranges.retrieve(r.key);
          expect(stored == null ? null : color.hex(stored)).toBe(hex);
        }
      });
    });
  });

  describe("copy", () => {
    it("should offer no time range for a range planned with no times", async () => {
      const planned = await createRange(uniqueRangeName("planned"), {
        timeRange: new TimeRange(TimeStamp.MAX, TimeStamp.MAX),
      });
      await renderMenu([planned]);
      const copy = await openSubmenu("Copy");
      within(copy).getByRole("menuitem", { name: "Name" });
      expect(within(copy).queryByRole("menuitem", { name: "Time range" })).toBeNull();
    });

    it("should copy the names of every range, one per line", async () => {
      const writeText = stubClipboardWriteText();
      const ranges = [
        await createRange(uniqueRangeName("copied")),
        await createRange(uniqueRangeName("copied")),
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
