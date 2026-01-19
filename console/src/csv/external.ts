// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DOWNLOAD_MODAL_LAYOUT_TYPE, DownloadModal } from "@/csv/DownloadModal";
import { type Layout } from "@/layout";

export * from "@/csv/DownloadModal";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [DOWNLOAD_MODAL_LAYOUT_TYPE]: DownloadModal,
};
