// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";

import { Editor } from "@/feature/arc/editor";
import { Explorer } from "@/feature/arc/explorer";
import { ingest } from "@/feature/arc/import";
import { Selectable } from "@/feature/arc/Selectable";
import { type Import } from "@/platform/import";
import { type Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";

export * from "@/feature/arc/commands";
export * from "@/feature/arc/editor";
export * from "@/feature/arc/explorer";
export * from "@/feature/arc/import";
export * from "@/feature/arc/link";
export * from "@/feature/arc/search";
export * from "@/feature/arc/toolbar/Toolbar";
export * from "@/feature/arc/tree";
export * from "@/platform/arc/external";

export const FILE_INGESTERS: Import.FileIngesters = {
  [arc.TYPE_ONTOLOGY_ID.type]: ingest,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];

export const TABS: Panel.Tabs = {
  ...Explorer.TABS,
  ...Editor.TABS,
};
