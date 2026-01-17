// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Connect, CONNECT_LAYOUT_TYPE } from "@/hardware/opc/device/Connect";
import { type Layout } from "@/layout";

export * from "@/hardware/opc/device/Browser";
export * from "@/hardware/opc/device/Connect";
export * from "@/hardware/opc/device/palette";
export * from "@/hardware/opc/device/Select";
export * from "@/hardware/opc/device/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CONNECT_LAYOUT_TYPE]: Connect,
};
