// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Connect, CONNECT_LAYOUT_TYPE } from "@/hardware/sift/device/Connect";
import { COMMANDS as PALETTE_COMMANDS } from "@/hardware/sift/device/palette";
import { type Layout } from "@/layout";
import { type Palette } from "@/palette";

export * from "@/hardware/sift/device/Connect";
export * from "@/hardware/sift/device/palette";
export * from "@/hardware/sift/device/types";

export const COMMANDS: Palette.Command[] = PALETTE_COMMANDS;

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CONNECT_LAYOUT_TYPE]: Connect,
};
