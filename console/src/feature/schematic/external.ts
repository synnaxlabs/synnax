// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";

import { extract } from "@/feature/schematic/export";
import { ingest } from "@/feature/schematic/import";
import { Schematic } from "@/feature/schematic/Schematic";
import { Selectable } from "@/feature/schematic/Selectable";
import { TabName } from "@/feature/schematic/TabName";
import { Toolbar } from "@/feature/schematic/toolbar/Toolbar";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";

export * from "@/feature/schematic/import";
export * from "@/feature/schematic/link";
export * from "@/feature/schematic/ontology";
export * from "@/feature/schematic/palette";
export * from "@/feature/schematic/symbol";
export * from "@/feature/schematic/toolbar/Toolbar";
export * from "@/platform/schematic/external";

const TAB_TYPE = schematic.TYPE_ONTOLOGY_ID.type;

export const EXTRACTORS: Export.Extractors = { [TAB_TYPE]: extract };

export const FILE_INGESTERS: Import.FileIngesters = { [TAB_TYPE]: ingest };

export const SELECTABLES: Selector.Selectable[] = [Selectable];

const TAB: Panel.Tab = {
  Content: Schematic,
  Toolbar,
  Name: TabName,
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};
