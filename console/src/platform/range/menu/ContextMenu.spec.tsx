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
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/platform/range";
import { uniqueRangeName } from "@/platform/range/testutil";
import {
  createConsoleWrapper,
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

    it("should leave the ranges nested in the ranges out of the list", async () => {
      const token = uniqueName("nested");
      const sibling = await createRange(`${token} sibling`);
      const self = await createRange(`${token} self`);
      const child = await createRange(`${token} child`, { parent: self.key });
      const grandchild = await createRange(`${token} grandchild`, {
        parent: child.key,
      });
      await renderMenu([self]);
      const parent = await openSubmenu("Parent");
      searchIn(parent, "Search ranges...", token);
      await within(parent).findByText(sibling.name);
      expect(within(parent).queryByText(child.name)).toBeNull();
      expect(within(parent).queryByText(grandchild.name)).toBeNull();
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
  });

  describe("copy", () => {
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
