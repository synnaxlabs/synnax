// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, query, schematic } from "@synnaxlabs/client";
import { Icon, Schematic as Base } from "@synnaxlabs/pluto";

import { ingest } from "@/feature/schematic/import";
import { Schematic } from "@/feature/schematic/Schematic";
import { Selectable } from "@/feature/schematic/Selectable";
import { Toolbar } from "@/feature/schematic/toolbar/Toolbar";
import { type Import } from "@/platform/import";
import { Panel } from "@/platform/panel";
import { type Range } from "@/platform/range";
import { type Selector } from "@/platform/selector";

export * from "@/feature/schematic/commands";
export * from "@/feature/schematic/import";
export * from "@/feature/schematic/link";
export * from "@/feature/schematic/search";
export * from "@/feature/schematic/symbol";
export * from "@/feature/schematic/toolbar/Toolbar";
export * from "@/feature/schematic/tree";
export * from "@/platform/schematic/external";

const TAB_TYPE = schematic.TYPE_ONTOLOGY_ID.type;

export const FILE_INGESTERS: Import.FileIngesters = { [TAB_TYPE]: ingest };

export const SELECTABLES: Selector.Selectable[] = [Selectable];

const TAB: Panel.Tab = {
  Content: Schematic,
  Toolbar,
  Icon: Icon.Schematic,
  Name: Panel.createEditableTabName(Base, <Icon.Schematic />),
  restore: async ({ client, project, resource }) => {
    const corpse = query.requireCorpse(client.schematics.getCached(resource.key));
    await client.schematics.create(project, corpse);
  },
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};

export const SNAPSHOT_SERVICES: Range.SnapshotServices = {
  [TAB_TYPE]: {
    icon: <Icon.Schematic />,
    onClick: async ({ id }, { client, openTab }) => {
      if (client == null) throw new DisconnectedError();
      await client.schematics.retrieve(id.key);
      openTab({ variant: "resource", resource: id });
    },
    onDelete: async ({ id: { key } }, { client }) => {
      if (client == null) throw new DisconnectedError();
      await client.schematics.delete(key);
    },
  },
};
