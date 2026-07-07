// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, group, ontology, ranger } from "@synnaxlabs/client";
import { fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { Range as CommonRange } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { createResource } from "@/platform/tree/testutil";
import { findTreeRow, renderOntologyTree } from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("range/ontology", () => {
  describe("onSelect", () => {
    it("adds the range to the session slice and places its overview", async () => {
      const rng = await createTestRange(client);
      const grp = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: uniqueName("rnggrp"),
      });
      await client.ontology.addChildren(
        group.ontologyID(grp.key),
        ranger.ontologyID(rng.key),
      );
      const { store } = await renderOntologyTree({
        client,
        root: group.ontologyID(grp.key),
        items: { range: Range.TREE_ITEM },
      });
      fireEvent.doubleClick(await findTreeRow(rng.name));
      await waitFor(() =>
        expect(Session.Range.selectState(store.getState(), rng.key)?.name).toBe(
          rng.name,
        ),
      );
      await waitFor(() =>
        expect(Session.Layout.select(store.getState(), rng.key)?.name).toBe(rng.name),
      );
      expect(Session.Layout.select(store.getState(), rng.key)?.type).toBe(
        CommonRange.OVERVIEW_LAYOUT_TYPE,
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
      const items = Range.TREE_ITEM.haulItems(resource);
      expect(items).toHaveLength(1);
      expect(items[0].key).toBe(rng.key);
    });

    it("returns nothing when the resource has no data payload", () => {
      const resource = createResource(ranger.ontologyID("some-key"), "no-data");
      expect(Range.TREE_ITEM.haulItems(resource)).toHaveLength(0);
    });
  });
});
