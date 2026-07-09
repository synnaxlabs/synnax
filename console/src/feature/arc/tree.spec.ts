// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc as clientArc, group, ontology } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { findTreeRow, renderOntologyTree } from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("arc/ontology", () => {
  describe("onSelect", () => {
    it("should place the arc editor layout when the row is double-clicked", async () => {
      const arc = await client.arcs.create({
        name: uniqueName("arc"),
        mode: "graph",
        graph: { nodes: [], edges: [] },
      });
      const grp = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: uniqueName("arcgrp"),
      });
      await client.ontology.addChildren(
        group.ontologyID(grp.key),
        clientArc.ontologyID(arc.key),
      );
      const { store } = await renderOntologyTree({
        client,
        root: group.ontologyID(grp.key),
        items: Arc.TREE_ITEMS,
      });
      fireEvent.doubleClick(await findTreeRow(arc.name));
      await waitFor(() =>
        expect(Session.Layout.select(store.getState(), arc.key)?.name).toBe(arc.name),
      );
      expect(Session.Layout.select(store.getState(), arc.key)?.type).toBe(
        Arc.EDITOR_LAYOUT_TYPE,
      );
    });
  });
});
