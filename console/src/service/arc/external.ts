// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Editor } from "@/component/arc/editor";
import { Explorer, EXPLORER_LAYOUT_TYPE } from "@/component/arc/explorer/Explorer";
import { type Export } from "@/export";
import { type Import } from "@/import";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";
import { extract } from "@/service/arc/imex/export";
import { ingest } from "@/service/arc/imex/import";
import { LAYOUT_TYPE } from "@/service/arc/layout";
import { Selectable } from "@/service/arc/Selectable";

export * from "@/component/arc/editor";
export * from "@/component/arc/explorer/Explorer";
export * from "@/service/arc/imex";
export * from "@/service/arc/layout";
export * from "@/service/arc/link";
export * from "@/service/arc/ontology";
export * from "@/service/arc/palette";
export * from "@/service/arc/toolbar/Toolbar";
export * from "@/service/arc/useCreate";
export * from "@/service/arc/useCreateModal";

export const EDITOR_LAYOUT_TYPE = LAYOUT_TYPE;
export type EditorLayoutType = typeof EDITOR_LAYOUT_TYPE;

export const EXTRACTORS: Export.Extractors = { [EDITOR_LAYOUT_TYPE]: extract };

export const FILE_INGESTERS: Import.FileIngesters = { [EDITOR_LAYOUT_TYPE]: ingest };

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [EXPLORER_LAYOUT_TYPE]: Explorer,
  [EDITOR_LAYOUT_TYPE]: Editor.Editor,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
