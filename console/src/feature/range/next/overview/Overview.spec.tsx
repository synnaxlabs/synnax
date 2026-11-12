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
  ranger,
  schematic,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Overview } from "@/feature/range/next/overview/Overview";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import {
  createPanelWrapper,
  createResourceTab,
  primePanel,
} from "@/platform/panel/testutil";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import {
  createTestClientWithGrants,
  type Grants,
  openContextMenu,
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

/** Creates a schematic snapshot that the range owns. */
const createDocument = async (rng: ranger.Range): Promise<schematic.Schematic> => {
  const proj = await client.projects.create({ name: uniqueName("proj") });
  const sch = await client.schematics.create({
    project: proj.key,
    name: uniqueName("sch"),
  });
  return await client.schematics.copy({
    key: sch.key,
    name: uniqueName("snap"),
    snapshot: true,
    range: rng.key,
  });
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
  describe("child ranges", () => {
    it("should show the children of a range that has not started on a timeline", async () => {
      const now = TimeStamp.now();
      const parent = await client.ranges.create({
        name: uniqueRangeName("planned"),
        timeRange: new TimeRange(now.add(TimeSpan.days(3)), TimeStamp.MAX),
      });
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

  describe("permissions", () => {
    const READS = [
      ranger.TYPE_RESOURCE_ID,
      label.TYPE_RESOURCE_ID,
      panel.TYPE_RESOURCE_ID,
      project.TYPE_RESOURCE_ID,
      schematic.TYPE_RESOURCE_ID,
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

    it("should offer a plot but no child range to a subject who may create only line plots", async () => {
      await renderAs({ create: [lineplot.TYPE_RESOURCE_ID] });
      await screen.findByRole("button", { name: "Plot" });
      expect(screen.queryByRole("button", { name: "Child range" })).toBeNull();
    });
  });
});
