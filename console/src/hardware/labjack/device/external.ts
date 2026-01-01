// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Configure, CONFIGURE_LAYOUT_TYPE } from "@/hardware/labjack/device/Configure";
import { type Layout } from "@/layout";

export * from "@/hardware/labjack/device/Configure";
export * from "@/hardware/labjack/device/Select";
export * from "@/hardware/labjack/device/SelectPort";
export * from "@/hardware/labjack/device/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CONFIGURE_LAYOUT_TYPE]: Configure,
};
