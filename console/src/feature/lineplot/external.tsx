// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, query } from "@synnaxlabs/client";
import { Icon, LinePlot as Base } from "@synnaxlabs/pluto";

import { ingest } from "@/feature/lineplot/import";
import { LinePlot } from "@/feature/lineplot/LinePlot";
import { Selectable } from "@/feature/lineplot/Selectable";
import { Toolbar } from "@/feature/lineplot/toolbar";
import { type Import } from "@/platform/import";
import { Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";

export * from "@/feature/lineplot/commands";
export * from "@/feature/lineplot/import";
export * from "@/feature/lineplot/LinePlot";
export * from "@/feature/lineplot/link";
export * from "@/feature/lineplot/search";
export * from "@/feature/lineplot/toolbar";
export * from "@/feature/lineplot/tree";
export * from "@/feature/lineplot/useTriggerHold";
export * from "@/platform/lineplot/external";

const TAB_TYPE = lineplot.TYPE_ONTOLOGY_ID.type;

export const FILE_INGESTERS: Import.FileIngesters = { [TAB_TYPE]: ingest };

export const SELECTABLES: Selector.Selectable[] = [Selectable];

const TAB: Panel.Tab = {
  Content: LinePlot,
  Toolbar,
  Icon: Icon.LinePlot,
  Name: Panel.createEditableTabName(Base, <Icon.LinePlot />),
  restore: async ({ client, project, resource }) => {
    const corpse = query.requireCorpse(client.lineplots.getCached(resource.key));
    await client.lineplots.create(project, corpse);
  },
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};
