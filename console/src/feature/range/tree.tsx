// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type ranger } from "@synnaxlabs/client";
import { type Haul, Icon, Ranger, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Layout } from "@/platform/layout";
import { Range } from "@/platform/range";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useOnSelect = (): ((resource: ontology.Resource) => void) => {
  const client = Synnax.use();
  const store = Session.useStore();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (resource) => {
      if (client == null) return;
      handleError(async () => {
        const ranges = await client.ranges.retrieve([resource.id.key]);
        store.dispatch(Session.Range.add(Session.Range.fromClient(ranges)));
        const first = ranges[0];
        placeLayout({ ...Range.OVERVIEW_LAYOUT, name: first.name, key: first.key });
      }, `Failed to select ${resource.name}`);
    },
    [client, store, placeLayout, handleError],
  );
};

const haulItems = (resource: ontology.Resource): Haul.Item[] => {
  const payload = resource.data as ranger.Payload | null | undefined;
  if (payload == null) return [];
  return [Ranger.createHaulItem(payload)];
};

const TreeItem = Tree.createItem({
  type: "range",
  icon: <Icon.Range />,
  useOnSelect,
  canDrop: () => true,
  haulItems,
});

export const TREE_ITEMS = { range: TreeItem } satisfies Tree.Items;
