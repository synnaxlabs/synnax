// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Editor } from "@/feature/arc/editor";
import { Explorer, EXPLORER_LAYOUT_TYPE } from "@/feature/arc/Explorer";
import { extract } from "@/feature/arc/export";
import { ingest } from "@/feature/arc/import";
import { Selectable } from "@/feature/arc/Selectable";
import { LAYOUT_TYPE } from "@/platform/arc/layout";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Layout } from "@/platform/layout";
import { type Selector } from "@/platform/selector";

export * from "@/feature/arc/editor";
export * from "@/feature/arc/Explorer";
export * from "@/feature/arc/export";
export * from "@/feature/arc/import";
export * from "@/feature/arc/link";
export * from "@/feature/arc/ontology";
export * from "@/feature/arc/palette";
export * from "@/feature/arc/toolbar/Toolbar";
export * from "@/platform/arc/layout";
export * from "@/platform/arc/useCreateModal";

export const EDITOR_LAYOUT_TYPE = LAYOUT_TYPE;
export type EditorLayoutType = typeof EDITOR_LAYOUT_TYPE;

export const EXTRACTORS: Export.Extractors = { [EDITOR_LAYOUT_TYPE]: extract };

export const FILE_INGESTERS: Import.FileIngesters = { [EDITOR_LAYOUT_TYPE]: ingest };

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [EXPLORER_LAYOUT_TYPE]: Explorer,
  [EDITOR_LAYOUT_TYPE]: Editor.Editor,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
export * from "@/platform/arc/useCreate";
