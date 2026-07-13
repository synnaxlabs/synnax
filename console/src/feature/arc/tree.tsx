// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type Synnax as Client } from "@synnaxlabs/client";
import { Icon, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Arc } from "@/platform/arc";
import { Layout } from "@/platform/layout";
import { Tree } from "@/platform/tree";

const load = async (client: Client, id: ontology.ID, placeLayout: Layout.Placer) => {
  const { name, key } = await client.arcs.retrieve({ key: id.key });
  placeLayout(Arc.create({ name, key }));
};

const useOnSelect = (): ((resource: ontology.Resource) => void) => {
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (resource) => {
      if (client == null) return;
      load(client, resource.id, placeLayout).catch((e: unknown) =>
        handleError(e, `Failed to load ${resource.name}`),
      );
    },
    [client, placeLayout, handleError],
  );
};

const TreeItem = Tree.createItem({
  type: "arc",
  icon: <Icon.Arc />,
  canDrop: () => true,
  useOnSelect,
});

export const TREE_ITEMS = { arc: TreeItem } satisfies Tree.Items;
