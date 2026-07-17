// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { type Flux, type Haul, Icon, Ranger, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Layout } from "@/platform/layout";
import { Range } from "@/platform/range";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useOnSelect = (): ((entry: Tree.Entry) => void) => {
  const client = Synnax.use();
  const store = Session.useStore();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (entry) => {
      if (client == null) return;
      handleError(async () => {
        const ranges = await client.ranges.retrieve([entry.id.key]);
        store.dispatch(Session.Range.add(Session.Range.fromClient(ranges)));
        const first = ranges[0];
        placeLayout({ ...Range.OVERVIEW_LAYOUT, name: first.name, key: first.key });
      }, `Failed to select ${entry.name}`);
    },
    [client, store, placeLayout, handleError],
  );
};

const haulItems = (resource: ontology.ID, store: Flux.Store): Haul.Item[] => {
  const range = (store as Ranger.FluxSubStore).ranges.get(resource.key);
  if (range == null) return [];
  return [Ranger.createHaulItem(range)];
};

const useName: Tree.UseName = (id) =>
  Ranger.useRetrieve({ key: id.key }).data?.name ?? "";

const TreeItem = Tree.createItem({
  type: "range",
  icon: <Icon.Range />,
  useName,
  useOnSelect,
  canDrop: () => true,
  haulItems,
});

export const TREE_ITEMS = { range: TreeItem } satisfies Tree.Items;
