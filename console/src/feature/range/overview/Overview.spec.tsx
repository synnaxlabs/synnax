// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  label,
  lineplot,
  NotFoundError,
  panel,
  project,
  query,
  ranger,
  schematic,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { type Flux } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { type ReactElement } from "react";
import { assert, describe, expect, it, onTestFinished } from "vitest";

import { Range } from "@/feature/range";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import {
  createPanelWrapper,
  createResourceTab,
  primePanel,
} from "@/platform/panel/testutil";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import { createCore, createCoreState } from "@/session/core/testutil";
import {
  createTestClientWithGrants,
  createTestStore,
  findTagCloseButton,
  type Grants,
  openContextMenu,
  renderSuspended,
  resolveFocusedTab,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const TAB = Range.Overview.TABS[ranger.TYPE_RESOURCE_ID.type];

const Deleted = ({ name }: Flux.Tombstone): ReactElement => <>{name} was deleted</>;

const CORE = createCore("overview");

interface RenderOverviewResult {
  store: TestStore;
  project: project.Key;
  /** Points the tab of the overview at the range. */
  setTabResource: (rangeKey: ranger.Key) => Promise<void>;
}

const renderOverview = async (
  rangeKey: ranger.Key,
  as: Client = client,
): Promise<RenderOverviewResult> => {
  const { panelKey, tabKey } = await createResourceTab(
    client,
    ranger.resourceID(rangeKey),
  );
  const proj = await client.projects.create({ name: uniqueName("proj") });
  const { wrapper, store } = await createPanelWrapper({
    client: as,
    store: await createTestStore({ preloadedState: createCoreState([CORE], CORE.key) }),
    panelKey,
    tabKey,
    project: proj.key,
  });
  await primePanel(wrapper, panelKey);
  const { Content, Guard } = TAB;
  assert(Guard != null);
  await renderSuspended(
    <>
      <Guard FallbackComponent={Deleted}>
        <Content />
      </Guard>
      <Modals.Stack />
    </>,
    { wrapper },
  );
  const setTabResource = async (next: ranger.Key): Promise<void> =>
    await act(async () => {
      await client.panels.dispatch(panelKey, [
        panel.setTabResource({ key: tabKey, resource: ranger.resourceID(next) }),
      ]);
    });
  return { store, project: proj.key, setTabResource };
};

const createChannel = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: uniqueName("ch"),
    dataType: "float32",
    virtual: true,
  });

const createChild = async (parent: ranger.Range): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await client.ranges.create({
    name: uniqueRangeName("child"),
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(1))),
    parent: parent.key,
  });
};

/** Creates the count of schematic snapshots that the range owns. */
const createDocuments = async (
  rng: ranger.Range,
  count: number,
): Promise<schematic.Schematic[]> => {
  const proj = await client.projects.create({ name: uniqueName("proj") });
  const sch = await client.schematics.create({
    project: proj.key,
    name: uniqueName("sch"),
  });
  return await Promise.all(
    Array.from(
      { length: count },
      async () =>
        await client.schematics.copy({
          key: sch.key,
          name: uniqueName("snap"),
          snapshot: true,
          range: rng.key,
        }),
    ),
  );
};

/** Creates a schematic snapshot that the range owns. */
const createDocument = async (rng: ranger.Range): Promise<schematic.Schematic> =>
  (await createDocuments(rng, 1))[0];

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

describe("range/overview", () => {
  describe("header", () => {
    it("should rename the range on the Core once the name loses focus", async () => {
      const rng = await createTestRange(client);
      await renderOverview(rng.key);
      const name = await screen.findByDisplayValue(rng.name);
      const renamed = uniqueRangeName("renamed");
      fireEvent.change(name, { target: { value: renamed } });
      fireEvent.blur(name);
      await waitFor(async () =>
        expect((await client.ranges.retrieve(rng.key)).name).toBe(renamed),
      );
    });

    it("should add a picked label to the range", async () => {
      const rng = await createTestRange(client);
      const lbl = await client.labels.create({
        name: uniqueName("label"),
        color: "#FF0000",
      });
      await renderOverview(rng.key);
      fireEvent.click(await screen.findByRole("button", { name: "Labels" }));
      fireEvent.change(await screen.findByPlaceholderText("Search labels..."), {
        target: { value: lbl.name },
      });
      fireEvent.click(await screen.findByRole("option", { name: lbl.name }));
      await waitFor(async () =>
        expect((await client.ranges.retrieve(rng.key)).labels).toEqual([lbl.key]),
      );
    });

    it("should remove a label when its tag closes", async () => {
      const rng = await createTestRange(client);
      const [removed, kept] = await client.labels.create([
        { name: uniqueName("removed"), color: "#FF0000" },
        { name: uniqueName("kept"), color: "#00FF00" },
      ]);
      await client.labels.label(rng.resourceID, [removed.key, kept.key]);
      await renderOverview(rng.key);
      await screen.findByText(removed.name);
      fireEvent.click(findTagCloseButton(removed.name));
      await waitFor(async () =>
        expect((await client.ranges.retrieve(rng.key)).labels).toEqual([kept.key]),
      );
    });

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

    it("should copy a link to the range", async () => {
      const writeText = stubClipboardWriteText();
      const rng = await createTestRange(client);
      await renderOverview(rng.key);
      fireEvent.click(await screen.findByRole("button", { name: "Copy link" }));
      await waitFor(() =>
        expect(writeText).toHaveBeenLastCalledWith(
          `synnax://cluster/${CORE.clusterKey}/range/${rng.key}`,
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

    it("should save an edited value", async () => {
      const rng = await createTestRange(client);
      await rng.kv.set("operator", "ada");
      await renderOverview(rng.key);
      const input = await screen.findByDisplayValue("ada");
      fireEvent.change(input, { target: { value: "grace" } });
      fireEvent.blur(input);
      await waitFor(async () => expect(await rng.kv.get("operator")).toBe("grace"));
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

  describe("aliases", () => {
    it("should add an alias to a channel", async () => {
      const rng = await createTestRange(client);
      const ch = await createChannel();
      await renderOverview(rng.key);
      fireEvent.click(await screen.findByRole("button", { name: "Channel alias" }));
      fireEvent.click(await screen.findByText("Channel"));
      fireEvent.change(await screen.findByPlaceholderText("Search channels..."), {
        target: { value: ch.name },
      });
      fireEvent.click(await screen.findByRole("option", { name: ch.name }));
      const alias = screen.getByPlaceholderText("Alias");
      fireEvent.change(alias, { target: { value: "thrust" } });
      fireEvent.keyDown(alias, { key: "Enter" });
      await screen.findByDisplayValue("thrust");
      await waitFor(async () =>
        expect(await rng.listAliases()).toEqual({ [ch.key]: "thrust" }),
      );
    });

    it("should note an alias that the parent range sets as inherited", async () => {
      const parent = await createTestRange(client);
      const child = await createChild(parent);
      const ch = await createChannel();
      await parent.setAlias(ch.key, "thrust");
      await renderOverview(child.key);
      await screen.findByDisplayValue("thrust");
      await screen.findByText("Inherited");
    });

    it("should let a range remove its own alias that equals the parent's", async () => {
      const parent = await createTestRange(client);
      const child = await createChild(parent);
      const ch = await createChannel();
      await parent.setAlias(ch.key, "thrust");
      await child.setAlias(ch.key, "thrust");
      // The overview of the parent was open, so the cache holds its aliases.
      await client.ranges.resolvedAliases.retrieve(parent.key);
      await renderOverview(child.key);
      await screen.findByDisplayValue("thrust");
      expect(screen.queryByText("Inherited")).toBeNull();
      fireEvent.click(screen.getByRole("button", { name: "Remove" }));
      await screen.findByText("Inherited");
    });
  });

  describe("attached", () => {
    it("should open a document in a tab", async () => {
      const rng = await createTestRange(client);
      const doc = await createDocument(rng);
      const { store } = await renderOverview(rng.key);
      fireEvent.click(await screen.findByText(doc.name));
      const tab = await resolveFocusedTab(
        store,
        client,
        (t) => t.variant === "resource",
      );
      if (tab.variant !== "resource") throw new Error("focused tab is not a resource");
      expect(tab.resource).toEqual(schematic.resourceID(doc.key));
    });

    it("should read the documents past the first page on Show more", async () => {
      const rng = await createTestRange(client);
      const docs = await createDocuments(rng, 51);
      await renderOverview(rng.key);
      fireEvent.click(await screen.findByRole("button", { name: "Show more" }));
      await waitFor(() => docs.forEach(({ name }) => screen.getByText(name)));
      expect(screen.queryByRole("button", { name: "Show more" })).toBeNull();
    });

    it("should delete a document after the confirm", async () => {
      const rng = await createTestRange(client);
      const doc = await createDocument(rng);
      await renderOverview(rng.key);
      await screen.findByText(doc.name);
      await openContextMenu(doc.name);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${doc.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(() => expect(screen.queryByText(doc.name)).toBeNull());
      await expect(client.schematics.retrieve({ key: doc.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("children", () => {
    const caret = (child: ranger.Range): Element | null =>
      screen
        .getByRole("listitem", { name: child.name })
        .querySelector(".console-range-item__caret");

    it("should drop the caret space when no child range has children", async () => {
      const rng = await createTestRange(client);
      await renderOverview(rng.key);
      const child = await createChild(rng);
      await screen.findByRole("listitem", { name: child.name });
      expect(caret(child)).toBeNull();
    });

    it("should bring the caret space back once a child range gets a child", async () => {
      const rng = await createTestRange(client);
      await renderOverview(rng.key);
      const parent = await createChild(rng);
      const leaf = await createChild(rng);
      await screen.findByRole("listitem", { name: leaf.name });
      expect(caret(leaf)).toBeNull();
      await createChild(parent);
      await waitFor(() => expect(caret(leaf)).not.toBeNull());
      expect(caret(parent)).not.toBeNull();
    });

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

  describe("tab", () => {
    it("should list the children of the range that the tab swaps to", async () => {
      const parent = await createTestRange(client);
      const child = await createChild(parent);
      const { setTabResource } = await renderOverview(child.key);
      await screen.findByDisplayValue(child.name);
      await setTabResource(parent.key);
      await screen.findByDisplayValue(parent.name);
      await screen.findByRole("listitem", { name: child.name });
    });

    it("should show the tombstone in place of the overview once the range is deleted", async () => {
      const rng = await createTestRange(client);
      await renderOverview(rng.key);
      await screen.findByDisplayValue(rng.name);
      await act(async () => await client.ranges.delete(rng.key));
      await screen.findByText(`${rng.name} was deleted`);
      expect(screen.queryByDisplayValue(rng.name)).toBeNull();
    });

    it("should show the overview again once the deleted range is restored", async () => {
      const { restore } = TAB;
      assert(restore != null);
      const rng = await createTestRange(client);
      const { project } = await renderOverview(rng.key);
      await screen.findByDisplayValue(rng.name);
      await act(async () => await client.ranges.delete(rng.key));
      await screen.findByText(`${rng.name} was deleted`);
      await act(
        async () => await restore({ client, project, resource: rng.resourceID }),
      );
      await screen.findByDisplayValue(rng.name);
    });

    it("should restore a deleted range under its original key", async () => {
      const { restore } = TAB;
      assert(restore != null);
      const rng = await createTestRange(client);
      // Restore rebuilds from the corpse the cache holds, which exists only once the
      // client has seen the range live.
      await client.ranges.retrieve(rng.key);
      await client.ranges.delete(rng.key);
      await waitFor(() =>
        expect(query.Deleted.matches(client.ranges.watch(rng.key).getCached())).toBe(
          true,
        ),
      );
      const proj = await client.projects.create({ name: uniqueName("proj") });
      await restore({ client, project: proj.key, resource: rng.resourceID });
      // Read back through a second client: the restoring client writes through to its
      // own cache, so a retrieve on it would pass even if the Core never got it.
      const remote = createTestClient();
      const [restored] = await remote.ranges.retrieve([rng.key]);
      expect(restored.name).toEqual(rng.name);
      expect(restored.timeRange).toEqual(rng.timeRange);
      // Name lookups read the range through the resource service.
      const named = await remote.resources.retrieve(rng.resourceID);
      expect(named.name).toEqual(rng.name);
    });
  });

  describe("permissions", () => {
    const READS = [
      ranger.TYPE_RESOURCE_ID,
      label.TYPE_RESOURCE_ID,
      panel.TYPE_RESOURCE_ID,
      project.TYPE_RESOURCE_ID,
      schematic.TYPE_RESOURCE_ID,
      lineplot.TYPE_RESOURCE_ID,
    ];

    const renderAs = async (grants: Grants): Promise<ranger.Range> => {
      const as = await createTestClientWithGrants(client, {
        ...grants,
        retrieve: READS,
      });
      const rng = await createTestRange(client);
      await renderOverview(rng.key, as);
      return rng;
    };

    it("should show the name as text to a subject who cannot update the range", async () => {
      const rng = await renderAs({});
      await screen.findByText(rng.name);
      expect(screen.queryByDisplayValue(rng.name)).toBeNull();
    });

    it("should offer a child range but no plot to a subject who may create only ranges", async () => {
      await renderAs({ create: [ranger.TYPE_RESOURCE_ID] });
      await screen.findByRole("button", { name: "Child range" });
      expect(screen.queryByRole("button", { name: "Plot" })).toBeNull();
    });

    it("should offer a plot but no child range to a subject who may create only line plots", async () => {
      await renderAs({ create: [lineplot.TYPE_RESOURCE_ID] });
      await screen.findByRole("button", { name: "Plot" });
      expect(screen.queryByRole("button", { name: "Child range" })).toBeNull();
    });
  });
});
