// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { findButton } from "@/platform/modals/testutil";
import { createTestRange } from "@/platform/range/testutil";
import { Tree } from "@/platform/tree";
import {
  createHostItems,
  createResource,
  expandTreeRow,
  findTreeRow,
  HOST_ID,
  openTreeRowContextMenu,
  renderLiveTree,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  awaitTextEditingElement,
  commitTextEdit,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const Item = Range.TREE_ITEMS.range;

const createHostedRange = async () => {
  const rng = await createTestRange(client);
  const items = {
    ...Range.TREE_ITEMS,
    ...createHostItems((c) => [
      Tree.createSource(c.ranges, { keys: [rng.key] }, (r) =>
        Tree.toResource("range")(r.payload),
      ),
    ]),
  };
  return { rng, items };
};

describe("range/tree", () => {
  describe("onSelect", () => {
    it("adds the range to the session slice and places its overview", async () => {
      const { rng, items } = await createHostedRange();
      const { store } = await renderLiveTree({
        client,
        root: HOST_ID,
        items,
      });
      fireEvent.doubleClick(await findTreeRow(rng.name));
      await waitFor(() =>
        expect(Session.Range.selectState(store.getState(), rng.key)?.variant).toBe(
          "persisted",
        ),
      );
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.key).toBe(rng.key);
    });
  });

  describe("context menu", () => {
    it("should offer Reload Console", async () => {
      const { rng, items } = await createHostedRange();
      await renderLiveTree({ client, root: HOST_ID, items });
      await openTreeRowContextMenu(rng.name);
      expect(await screen.findByText("Reload Console")).toBeTruthy();
    });

    it("renames the range in place and syncs the favorited copy", async () => {
      const { rng, items } = await createHostedRange();
      const { store } = await renderLiveTree({
        client,
        root: HOST_ID,
        items,
      });
      store.dispatch(Session.Range.add(Session.Range.fromClient(rng.payload)));
      await openTreeRowContextMenu(rng.name);
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditingElement();
      const renamed = uniqueName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(async () =>
        expect((await client.ranges.retrieve(rng.key)).name).toBe(renamed),
      );
    });

    it("drops the favorited copy when the range is deleted", async () => {
      const { rng, items } = await createHostedRange();
      const { store } = await renderLiveTree({
        client,
        root: HOST_ID,
        items,
      });
      store.dispatch(Session.Range.add(Session.Range.fromClient(rng.payload)));
      await openTreeRowContextMenu(rng.name);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${rng.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(async () => {
        await expect(client.ranges.retrieve(rng.key)).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
      await waitFor(() =>
        expect(Session.Range.selectState(store.getState(), rng.key)).toBeUndefined(),
      );
    });
  });

  describe("children", () => {
    it("should list the ranges nested in the range", async () => {
      const { rng: parent, items } = await createHostedRange();
      const child = await client.ranges.create({
        name: uniqueName("child"),
        timeRange: parent.timeRange,
        parent: parent.key,
      });
      await renderLiveTree({ client, root: HOST_ID, items });
      await findTreeRow(parent.name);
      expandTreeRow(parent.name);
      expect(await findTreeRow(child.name)).toBeTruthy();
    });
  });

  describe("haulItems", () => {
    it("returns a range haul item carrying the resource payload", async () => {
      const rng = await createTestRange(client);
      const resource = createResource(ranger.resourceID(rng.key), rng.name, {
        key: rng.key,
        name: rng.name,
        timeRange: rng.timeRange,
      });
      const items = Item.haulItems(resource);
      expect(items).toHaveLength(1);
      expect(items[0].key).toBe(rng.key);
    });

    it("returns nothing when the resource has no data payload", () => {
      const resource = createResource(ranger.resourceID("some-key"), "no-data");
      expect(Item.haulItems(resource)).toHaveLength(0);
    });
  });
});
