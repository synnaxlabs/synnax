// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, query } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Arc } from "@synnaxlabs/pluto";

import { Panel } from "@/platform/panel";

const TAB: Panel.Tab = {
  Content: Panel.lazyComponent(() => import("@/feature/arc/editor/Editor"), "Editor"),
  Toolbar: Panel.lazyComponent(
    () => import("@/feature/arc/editor/toolbar/Toolbar"),
    "Toolbar",
  ),
  Icon: Icon.Arc,
  // Passing the whole namespace would keep all of its code in the startup bundle.
  Name: Panel.createEditableTabName(
    { useEnsure: Arc.useEnsure, useName: Arc.useName, useRename: Arc.useRename },
    <Icon.Arc />,
  ),
  restore: async ({ client, resource }) => {
    const corpse = query.requireCorpse(client.arcs.getCached(resource.key));
    await client.arcs.create(corpse);
  },
  // Passing the whole namespace would keep all of its code in the startup bundle.
  useTombstone: Panel.createTombstoneReader({ useTombstone: Arc.useTombstone }),
};

export const TABS: Panel.Tabs = {
  [arc.TYPE_ONTOLOGY_ID.type]: TAB,
};
