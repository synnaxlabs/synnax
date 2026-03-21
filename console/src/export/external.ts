// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EXPORT_LAYOUT_TYPE, ExportModal } from "@/export/ExportModal";
import { COMMANDS as PALETTE_COMMANDS } from "@/export/palette";
import { type Layout } from "@/layout";
import { type Palette } from "@/palette";

export * from "@/export/download";
export { EXPORT_LAYOUT, EXPORT_LAYOUT_TYPE, ExportModal } from "@/export/ExportModal";
export * from "@/export/extractor";
export * from "@/export/ExtractorsProvider";
export * from "@/export/MenuItem";
export * from "@/export/ToolbarButton";
export * from "@/export/use";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [EXPORT_LAYOUT_TYPE]: ExportModal,
};

export const COMMANDS: Palette.Command[] = [...PALETTE_COMMANDS];
