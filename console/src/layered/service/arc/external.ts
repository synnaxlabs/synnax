// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { type Import } from "@/import";
import { Editor } from "@/layered/service/arc/editor";
import { Explorer, EXPLORER_LAYOUT_TYPE } from "@/layered/service/arc/Explorer";
import { extract } from "@/layered/service/arc/imex/export";
import { ingest } from "@/layered/service/arc/imex/import";
import { LAYOUT_TYPE } from "@/layered/service/arc/layout";
import { Selectable } from "@/layered/service/arc/Selectable";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/layered/service/arc/CreateModal";
export * from "@/layered/service/arc/editor";
export * from "@/layered/service/arc/Explorer";
export * from "@/layered/service/arc/imex";
export * from "@/layered/service/arc/layout";
export * from "@/layered/service/arc/link";
export * from "@/layered/service/arc/ontology";
export * from "@/layered/service/arc/palette";
export * from "@/layered/service/arc/toolbar/Toolbar";
export * from "@/layered/service/arc/useCreate";

export const EDITOR_LAYOUT_TYPE = LAYOUT_TYPE;
export type EditorLayoutType = typeof EDITOR_LAYOUT_TYPE;

export const EXTRACTORS: Export.Extractors = { [EDITOR_LAYOUT_TYPE]: extract };

export const FILE_INGESTERS: Import.FileIngesters = { [EDITOR_LAYOUT_TYPE]: ingest };

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [EXPLORER_LAYOUT_TYPE]: Explorer,
  [EDITOR_LAYOUT_TYPE]: Editor.Editor,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
