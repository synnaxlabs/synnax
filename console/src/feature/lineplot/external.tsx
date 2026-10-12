// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, query } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { LinePlot as Base } from "@synnaxlabs/pluto";

import { Selectable } from "@/feature/lineplot/Selectable";
import { Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";

export * from "@/feature/lineplot/commands";
export * from "@/feature/lineplot/link";
export * from "@/feature/lineplot/search";
export * from "@/feature/lineplot/tree";
export * from "@/platform/lineplot/external";

const TAB_TYPE = lineplot.TYPE_ONTOLOGY_ID.type;

export const SELECTABLES: Selector.Selectable[] = [Selectable];

const TAB: Panel.Tab = {
  Content: Panel.lazyComponent(() => import("@/feature/lineplot/LinePlot"), "LinePlot"),
  Toolbar: Panel.lazyComponent(() => import("@/feature/lineplot/toolbar"), "Toolbar"),
  Icon: Icon.LinePlot,
  // Only these hooks, so the bundler can drop the rest of the namespace.
  Name: Panel.createEditableTabName(
    { useEnsure: Base.useEnsure, useName: Base.useName, useRename: Base.useRename },
    <Icon.LinePlot />,
  ),
  restore: async ({ client, project, resource }) => {
    const corpse = query.requireCorpse(client.lineplots.getCached(resource.key));
    await client.lineplots.create(project, corpse);
  },
  // Only these hooks, so the bundler can drop the rest of the namespace.
  useTombstone: Panel.createTombstoneReader({ useTombstone: Base.useTombstone }),
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};
