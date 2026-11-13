// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  label,
  lineplot,
  panel,
  project,
  ranger,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { color, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, onTestFinished } from "vitest";

import { Overview } from "@/feature/range/next/overview/Overview";
import { Modals } from "@/platform/modals";
import {
  createPanelWrapper,
  createResourceTab,
  primePanel,
} from "@/platform/panel/testutil";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import {
  createTestClientWithGrants,
  type Grants,
  queryIconButton,
  renderSuspended,
  resolveFocusedTab,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const renderOverview = async (
  rangeKey: ranger.Key,
  as: Client = client,
): Promise<{ store: TestStore }> => {
  const { panelKey, tabKey } = await createResourceTab(
    client,
    ranger.resourceID(rangeKey),
  );
  const proj = await client.projects.create({ name: uniqueName("proj") });
  const { wrapper, store } = await createPanelWrapper({
    client: as,
    panelKey,
    tabKey,
    project: proj.key,
  });
  await primePanel(wrapper, panelKey);
  await renderSuspended(
    <>
      <Overview />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { store };
};

const createChild = async (parent: ranger.Range): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await client.ranges.create({
    name: uniqueRangeName("child"),
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(1))),
    parent: parent.key,
  });
};

/** @returns the text of the tooltip that a hover on the element shows. */
const hoverTip = async (el: HTMLElement): Promise<string> => {
  fireEvent.pointerOver(el, { pointerType: "mouse" });
  return (await screen.findByRole("tooltip")).textContent;
};

/** Creates a range under the parent that starts at the start and runs one second. */
const createChildAt = async (
  parent: ranger.Range,
  start: TimeStamp,
): Promise<ranger.Range> =>
  await client.ranges.create({
    name: uniqueRangeName("child"),
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(1))),
    parent: parent.key,
  });

describe("range/next/overview", () => {
  describe("header", () => {
    it("should show the ranges above as crumbs and open one on click", async () => {
      const top = await createTestRange(client);
      const parent = await createChild(top);
      const rng = await createChild(parent);
      const { store } = await renderOverview(rng.key);
      await screen.findByRole("button", { name: parent.name });
      const crumbs = screen
        .getAllByRole("button")
        .filter((el) => [top.name, parent.name].includes(el.textContent ?? ""));
      expect(crumbs.map((el) => el.textContent)).toEqual([top.name, parent.name]);
      fireEvent.click(screen.getByRole("button", { name: parent.name }));
      const tab = await resolveFocusedTab(
        store,
        client,
        (t) => t.variant === "resource" && t.resource.key === parent.key,
      );
      if (tab.variant !== "resource") throw new Error("focused tab is not a resource");
      expect(tab.resource).toEqual(ranger.resourceID(parent.key));
    });

    it("should set the color of the range from its swatch", async () => {
      const rng = await createTestRange(client);
      await renderOverview(rng.key);
      fireEvent.click(await screen.findByRole("button", { name: "Color" }));
      const hex = await screen.findByLabelText("Hex");
      fireEvent.change(hex, { target: { value: "123456" } });
      fireEvent.keyDown(hex, { code: "Escape" });
      await waitFor(async () => {
        const { color: stored } = await client.ranges.retrieve(rng.key);
        expect(stored == null ? null : color.hex(stored)).toBe("#123456");
      });
    });

    it("should copy Python and TypeScript code that retrieves the range", async () => {
      const writeText = stubClipboardWriteText();
      const rng = await createTestRange(client);
      await renderOverview(rng.key);
      fireEvent.click(await screen.findByRole("button", { name: "Copy Python code" }));
      await waitFor(() =>
        expect(writeText).toHaveBeenLastCalledWith(
          `rng = client.ranges.retrieve(key="${rng.key}")`,
        ),
      );
      fireEvent.click(screen.getByRole("button", { name: "Copy TypeScript code" }));
      await waitFor(() =>
        expect(writeText).toHaveBeenLastCalledWith(
          `const rng = await client.ranges.retrieve("${rng.key}");`,
        ),
      );
    });

    it("should open the CSV download for the range", async () => {
      const rng = await createTestRange(client);
      await renderOverview(rng.key);
      fireEvent.click(await screen.findByRole("button", { name: "Download as CSV" }));
      await screen.findByText(`Download data for ${rng.name} as CSV`);
    });
  });

  describe("child ranges", () => {
    it("should show the children of a range that has not started on a timeline", async () => {
      const now = TimeStamp.now();
      const parent = await client.ranges.create({
        name: uniqueRangeName("planned"),
        timeRange: new TimeRange(now.add(TimeSpan.days(3)), TimeStamp.MAX),
      });
      // A range that has not started sorts above every other in the explorer.
      onTestFinished(async () => await client.ranges.delete(parent.key));
      const child = await createChildAt(parent, now);
      await renderOverview(parent.key, createTestClient());
      await screen.findByRole("listitem", { name: child.name });
      expect(screen.getByRole("button", { name: "Now" })).toBeTruthy();
    });

    it("should list the children of an open child newest first in the tree", async () => {
      const end = TimeStamp.now();
      const parent = await client.ranges.create({
        name: uniqueRangeName("campaign"),
        timeRange: new TimeRange(end.sub(TimeSpan.days(2)), end),
      });
      const child = await createChildAt(parent, end.sub(TimeSpan.hours(2)));
      const older = await createChildAt(child, end.sub(TimeSpan.hours(2)));
      const newer = await createChildAt(child, end.sub(TimeSpan.hours(1)));
      // A fresh client reads that the parent has children.
      await renderOverview(parent.key, createTestClient());
      const item = await screen.findByRole("listitem", { name: child.name });
      fireEvent.click(await within(item).findByRole("button", { name: "Expand" }));
      await screen.findByRole("listitem", { name: older.name });
      const names = screen
        .getAllByRole("listitem")
        .map((el) => el.getAttribute("aria-label"));
      expect(names).toEqual([child.name, newer.name, older.name]);
    });
  });

  describe("metadata", () => {
    it("should add a pair from the add line of the empty section", async () => {
      const rng = await createTestRange(client);
      await renderOverview(rng.key);
      fireEvent.click(await screen.findByRole("button", { name: "Metadata" }));
      fireEvent.change(await screen.findByPlaceholderText("Key"), {
        target: { value: "operator" },
      });
      const value = screen.getByPlaceholderText("Value");
      fireEvent.change(value, { target: { value: "ada" } });
      fireEvent.keyDown(value, { key: "Enter" });
      await waitFor(() => expect(screen.queryByPlaceholderText("Key")).toBeNull());
      expect(screen.getByText("operator")).toBeTruthy();
      expect(screen.getByDisplayValue("ada")).toBeTruthy();
      expect(await rng.kv.get("operator")).toBe("ada");
    });

    it("should mark a blank key and keep the draft open on Enter", async () => {
      const rng = await createTestRange(client);
      await renderOverview(rng.key);
      fireEvent.click(await screen.findByRole("button", { name: "Metadata" }));
      const value = await screen.findByPlaceholderText("Value");
      fireEvent.change(value, { target: { value: "ada" } });
      fireEvent.keyDown(value, { key: "Enter" });
      const key = screen.getByPlaceholderText("Key");
      await waitFor(() => expect(key.getAttribute("aria-invalid")).toBe("true"));
      expect(await hoverTip(key)).toBe("Key is required");
      expect(await rng.kv.list()).toEqual({});
    });

    it("should save an edited value", async () => {
      const rng = await createTestRange(client);
      await rng.kv.set("operator", "ada");
      await renderOverview(rng.key);
      const input = await screen.findByDisplayValue("ada");
      fireEvent.change(input, { target: { value: "grace" } });
      fireEvent.blur(input);
      await waitFor(async () => expect(await rng.kv.get("operator")).toBe("grace"));
    });

    it("should keep a value that another client sets while the value has focus", async () => {
      const rng = await createTestRange(client);
      await rng.kv.set("operator", "ada");
      await renderOverview(rng.key);
      const input = await screen.findByDisplayValue("ada");
      fireEvent.focus(input);
      await rng.kv.set("operator", "grace");
      // The changes land in order, so the second pair shows that the first landed.
      await rng.kv.set("shift", "night");
      await screen.findByDisplayValue("night");
      fireEvent.blur(input);
      await screen.findByDisplayValue("grace");
      expect(await rng.kv.get("operator")).toBe("grace");
    });

    it("should remove a pair", async () => {
      const rng = await createTestRange(client);
      await rng.kv.set("operator", "ada");
      await renderOverview(rng.key);
      await screen.findByDisplayValue("ada");
      fireEvent.click(screen.getByRole("button", { name: "Remove" }));
      await waitFor(() => expect(screen.queryByDisplayValue("ada")).toBeNull());
      await waitFor(async () => expect(await rng.kv.list()).toEqual({}));
    });

    it("should show a value that another client sets", async () => {
      const rng = await createTestRange(client);
      await renderOverview(rng.key);
      await screen.findByRole("button", { name: "Metadata" });
      const other = await createTestClient().ranges.retrieve(rng.key);
      await other.kv.set("operator", "ada");
      await screen.findByDisplayValue("ada");
    });
  });

  describe("permissions", () => {
    const READS = [
      ranger.TYPE_RESOURCE_ID,
      label.TYPE_RESOURCE_ID,
      panel.TYPE_RESOURCE_ID,
      project.TYPE_RESOURCE_ID,
      lineplot.TYPE_RESOURCE_ID,
    ];

    const renderAs = async (grants: Grants): Promise<void> => {
      const as = await createTestClientWithGrants(client, {
        ...grants,
        retrieve: READS,
      });
      const rng = await createTestRange(client);
      await renderOverview(rng.key, as);
    };

    it("should offer a child range but no plot to a subject who may create only ranges", async () => {
      await renderAs({ create: [ranger.TYPE_RESOURCE_ID] });
      await screen.findByRole("button", { name: "Child range" });
      expect(screen.queryByRole("button", { name: "Plot" })).toBeNull();
    });

    it("should show metadata without edit controls to a subject who may not update the range", async () => {
      const rng = await createTestRange(client);
      await rng.kv.set("operator", "ada");
      const as = await createTestClientWithGrants(client, {
        create: [lineplot.TYPE_RESOURCE_ID],
        retrieve: READS,
      });
      await renderOverview(rng.key, as);
      // Both grants answer from one policy read, so this button shows that it landed.
      await screen.findByRole("button", { name: "Plot" });
      await screen.findByText("ada");
      expect(screen.queryByDisplayValue("ada")).toBeNull();
      expect(screen.queryByRole("button", { name: "Remove" })).toBeNull();
      expect(queryIconButton(document.body, "add")).toBeNull();
    });

    it("should offer a plot but no child range to a subject who may create only line plots", async () => {
      await renderAs({ create: [lineplot.TYPE_RESOURCE_ID] });
      await screen.findByRole("button", { name: "Plot" });
      expect(screen.queryByRole("button", { name: "Child range" })).toBeNull();
    });
  });
});
