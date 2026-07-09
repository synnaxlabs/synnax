// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type ontology, type Synnax as Client } from "@synnaxlabs/client";
import { Icon, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";

const load = async (client: Client, id: ontology.ID, openTab: Panel.OpenTab) => {
  const { key } = await client.arcs.retrieve({ key: id.key });
  openTab({ variant: "resource", resource: arc.ontologyID(key) });
};

const useOnSelect = (): ((resource: ontology.Resource) => void) => {
  const client = Synnax.use();
  const openTab = Panel.useOpenTab();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (resource) => {
      if (client == null) return;
      load(client, resource.id, openTab).catch((e: unknown) =>
        handleError(e, `Failed to load ${resource.name}`),
      );
    },
    [client, openTab, handleError],
  );
};

const TreeItem = Tree.createItem({
  type: "arc",
  icon: <Icon.Arc />,
  canDrop: () => true,
  useOnSelect,
});

export const TREE_ITEMS = { arc: TreeItem } satisfies Tree.Items;
