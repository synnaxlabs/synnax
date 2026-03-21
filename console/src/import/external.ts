// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { IMPORT_LAYOUT_TYPE, ImportModal } from "@/import/ImportModal";
import { type Layout } from "@/layout";

export * from "@/import/dataTransferItem";
export * from "@/import/FileIngestersProvider";
export * from "@/import/import";
export { IMPORT_LAYOUT, IMPORT_LAYOUT_TYPE, ImportModal } from "@/import/ImportModal";
export * from "@/import/ingester";
export * from "@/import/palette";
export * from "@/import/trimFileName";
export * from "@/import/upload";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [IMPORT_LAYOUT_TYPE]: ImportModal,
};
