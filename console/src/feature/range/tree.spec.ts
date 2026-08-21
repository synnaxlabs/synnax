// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, NotFoundError, ontology, ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { findButton } from "@/platform/modals/testutil";
import { createTestRange } from "@/platform/range/testutil";
import { createResource } from "@/platform/tree/testutil";
import {
  findTreeRow,
  openTreeRowContextMenu,
  renderOntologyTree,
} from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import {
  awaitTextEditingElement,
  commitTextEdit,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const Item = Range.TREE_ITEMS.range;

const createRangeInGroup = async () => {
  const rng = await createTestRange(client);
  const grp = await client.groups.create({
    parent: ontology.ROOT_ID,
    name: uniqueName("rnggrp"),
  });
  await client.ontology.addChildren(
    group.ontologyID(grp.key),
    ranger.ontologyID(rng.key),
  );
  return { rng, root: group.ontologyID(grp.key) };
};

describe("range/ontology", () => {
  describe("onSelect", () => {
    it("adds the range to the session slice and places its overview", async () => {
      const { rng, root } = await createRangeInGroup();
      const { store } = await renderOntologyTree({
        client,
        root,
        items: Range.TREE_ITEMS,
      });
      fireEvent.doubleClick(await findTreeRow(rng.name));
      await waitFor(() =>
        expect(Session.Range.selectState(store.getState(), rng.key)?.name).toBe(
          rng.name,
        ),
      );
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.key).toBe(rng.key);
    });
  });

  describe("context menu", () => {
    it("renames the range in place and syncs the favorited copy", async () => {
      const { rng, root } = await createRangeInGroup();
      const { store } = await renderOntologyTree({
        client,
        root,
        items: Range.TREE_ITEMS,
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
      await waitFor(() =>
        expect(Session.Range.selectState(store.getState(), rng.key)?.name).toBe(
          renamed,
        ),
      );
    });

    it("drops the favorited copy when the range is deleted", async () => {
      const { rng, root } = await createRangeInGroup();
      const { store } = await renderOntologyTree({
        client,
        root,
        items: Range.TREE_ITEMS,
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

  describe("haulItems", () => {
    it("returns a range haul item carrying the resource payload", async () => {
      const rng = await createTestRange(client);
      const resource = createResource(ranger.ontologyID(rng.key), rng.name, {
        key: rng.key,
        name: rng.name,
        timeRange: rng.timeRange,
      });
      const items = Item.haulItems(resource);
      expect(items).toHaveLength(1);
      expect(items[0].key).toBe(rng.key);
    });

    it("returns nothing when the resource has no data payload", () => {
      const resource = createResource(ranger.ontologyID("some-key"), "no-data");
      expect(Item.haulItems(resource)).toHaveLength(0);
    });
  });
});
