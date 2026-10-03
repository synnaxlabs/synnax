// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, table } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Table as Base } from "@synnaxlabs/pluto";

import { Selectable } from "@/feature/table/Selectable";
import { Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";

export * from "@/feature/table/commands";
export * from "@/feature/table/link";
export * from "@/feature/table/search";
export * from "@/feature/table/tree";
export * from "@/platform/table/external";

const TAB_TYPE = table.TYPE_ONTOLOGY_ID.type;

export const SELECTABLES: Selector.Selectable[] = [Selectable];

const TAB: Panel.Tab = {
  Content: Panel.lazyComponent(() => import("@/feature/table/Table"), "Table"),
  Toolbar: Panel.lazyComponent(() => import("@/feature/table/Toolbar"), "Toolbar"),
  Icon: Icon.Table,
  // Only these hooks, so the bundler can drop the rest of the namespace.
  Name: Panel.createEditableTabName(
    { useEnsure: Base.useEnsure, useName: Base.useName, useRename: Base.useRename },
    <Icon.Table />,
  ),
  restore: async ({ client, project, resource }) => {
    const corpse = query.requireCorpse(client.tables.getCached(resource.key));
    await client.tables.create(project, corpse);
  },
  // Only these hooks, so the bundler can drop the rest of the namespace.
  useTombstone: Panel.createTombstoneReader({ useTombstone: Base.useTombstone }),
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};
