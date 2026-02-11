// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Configure, CONFIGURE_LAYOUT_TYPE } from "@/hardware/ethercat/device/Configure";
import { type Layout } from "@/layout";

export * from "@/hardware/ethercat/device/Configure";
export * from "@/hardware/ethercat/device/queries";
export * from "@/hardware/ethercat/device/SelectSlave";
export * from "@/hardware/ethercat/device/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CONFIGURE_LAYOUT_TYPE]: Configure,
};
