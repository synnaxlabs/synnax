// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Selector, SELECTOR_LAYOUT_TYPE } from "@/hardware/task/Selector";
import { type Layout } from "@/layout";
import { type Palette } from "@/palette";

export * from "@/hardware/task/link";
export * from "@/hardware/task/ontology";
export * from "@/hardware/task/Selector";
export * from "@/hardware/task/Toolbar";
export * from "@/hardware/task/types";

export const COMMANDS: Palette.Command[] = [
  ...LabJack.Task.COMMANDS,
  ...NI.Task.COMMANDS,
  ...OPC.Task.COMMANDS,
];

export const LAYOUTS: Record<string, Layout.Renderer> = {
  ...LabJack.Task.LAYOUTS,
  ...NI.Task.LAYOUTS,
  ...OPC.Task.LAYOUTS,
  [SELECTOR_LAYOUT_TYPE]: Selector,
};
