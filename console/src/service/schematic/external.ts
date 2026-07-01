// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/component/layout";
import { ContextMenu } from "@/component/schematic/ContextMenu";
import { LAYOUT_TYPE } from "@/component/schematic/layout";
import { Schematic } from "@/component/schematic/Schematic";
import { type Selector } from "@/component/selector";
import { type Export } from "@/service/export";
import { extract } from "@/service/schematic/export";
import { FILE_INGESTERS } from "@/service/schematic/import";
import { Selectable } from "@/service/schematic/Selectable";

export * from "@/component/schematic/layout";
export * from "@/component/schematic/useCreate";
export * from "@/service/schematic/link";
export * from "@/service/schematic/ontology";
export * from "@/service/schematic/palette";
export * from "@/service/schematic/toolbar/Toolbar";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const ImEx = { FILE_INGESTERS };

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Schematic,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
