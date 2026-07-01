// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/feature/export";
import { ContextMenu } from "@/feature/schematic/ContextMenu";
import { extract } from "@/feature/schematic/export";
import { FILE_INGESTERS } from "@/feature/schematic/import";
import { Schematic } from "@/feature/schematic/Schematic";
import { Selectable } from "@/feature/schematic/Selectable";
import { type Layout } from "@/platform/layout";
import { LAYOUT_TYPE } from "@/platform/schematic/layout";
import { type Selector } from "@/platform/selector";

export * from "@/feature/schematic/link";
export * from "@/feature/schematic/ontology";
export * from "@/feature/schematic/palette";
export * from "@/feature/schematic/toolbar/Toolbar";
export * from "@/platform/schematic/layout";
export * from "@/platform/schematic/useCreate";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const ImEx = { FILE_INGESTERS };

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Schematic,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
