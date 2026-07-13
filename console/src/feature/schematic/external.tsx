// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { Icon, Schematic as PSchematic } from "@synnaxlabs/pluto";

import { ContextMenu } from "@/feature/schematic/ContextMenu";
import { extract } from "@/feature/schematic/export";
import { ingest } from "@/feature/schematic/import";
import { Schematic } from "@/feature/schematic/Schematic";
import { Selectable } from "@/feature/schematic/Selectable";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Layout } from "@/platform/layout";
import { type Range } from "@/platform/range";
import { Schematic as PlatformSchematic } from "@/platform/schematic";
import { type Selector } from "@/platform/selector";

export * from "@/feature/schematic/commands";
export * from "@/feature/schematic/import";
export * from "@/feature/schematic/link";
export * from "@/feature/schematic/search";
export * from "@/feature/schematic/symbol";
export * from "@/feature/schematic/toolbar/Toolbar";
export * from "@/feature/schematic/tree";
export * from "@/feature/schematic/useMosaicDrop";
export * from "@/platform/schematic/external";

export const CONTEXT_MENUS: Layout.ContextMenuRenderers = {
  [PlatformSchematic.LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = {
  [PlatformSchematic.LAYOUT_TYPE]: extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [PlatformSchematic.LAYOUT_TYPE]: ingest,
};

export const LAYOUTS: Layout.Renderers = {
  [PlatformSchematic.LAYOUT_TYPE]: Schematic,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];

export const SNAPSHOT_SERVICES: Range.SnapshotServices = {
  [PlatformSchematic.LAYOUT_TYPE]: {
    icon: <Icon.Schematic />,
    useIsSnapshot: (key) => PSchematic.useRetrieve({ key }).data?.snapshot === true,
    onClick: async ({ id: { key } }, { client, placeLayout }) => {
      if (client == null) throw new DisconnectedError();
      const s = await client.schematics.retrieve({ key });
      placeLayout(
        PlatformSchematic.create({ key: s.key, name: s.name, editable: false }),
      );
    },
    onDelete: async ({ id: { key } }, { client }) => {
      if (client == null) throw new DisconnectedError();
      await client.schematics.delete(key);
    },
  },
};
