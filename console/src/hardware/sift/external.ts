// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device } from "@/hardware/sift/device";
import {
  EXPORT_MODAL_LAYOUT_TYPE,
  ExportModal,
} from "@/hardware/sift/ExportModal";
import { type Layout } from "@/layout";

export * from "@/hardware/sift/device";
export * from "@/hardware/sift/ExportModal";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  ...Device.LAYOUTS,
  [EXPORT_MODAL_LAYOUT_TYPE]: ExportModal,
};
