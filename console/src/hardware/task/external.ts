// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { EtherCAT } from "@/hardware/ethercat";
import { LabJack } from "@/hardware/labjack";
import { Modbus } from "@/hardware/modbus";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Selector, SELECTOR_LAYOUT_TYPE } from "@/hardware/task/Selector";
import { Sequence } from "@/hardware/task/sequence";
import { TOOLBAR_NAV_DRAWER_ITEM } from "@/hardware/task/Toolbar";
import { type Import } from "@/import";
import { type Layout } from "@/layout";
import { type Palette } from "@/palette";

export * from "@/hardware/task/layouts";
export * from "@/hardware/task/link";
export * from "@/hardware/task/ontology";
export * from "@/hardware/task/Selector";
export * from "@/hardware/task/Toolbar";

export const COMMANDS: Palette.Command[] = [
  ...EtherCAT.Task.COMMANDS,
  ...LabJack.Task.COMMANDS,
  ...Modbus.Task.COMMANDS,
  ...NI.Task.COMMANDS,
  ...OPC.Task.COMMANDS,
  ...Sequence.COMMANDS,
];

export const EXTRACTORS: Export.Extractors = {
  ...EtherCAT.Task.EXTRACTORS,
  ...LabJack.Task.EXTRACTORS,
  ...Modbus.Task.EXTRACTORS,
  ...NI.Task.EXTRACTORS,
  ...OPC.Task.EXTRACTORS,
  ...Sequence.EXTRACTORS,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  ...EtherCAT.Task.FILE_INGESTERS,
  ...LabJack.Task.FILE_INGESTERS,
  ...Modbus.Task.FILE_INGESTERS,
  ...NI.Task.FILE_INGESTERS,
  ...OPC.Task.FILE_INGESTERS,
  ...Sequence.FILE_INGESTERS,
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  ...EtherCAT.Task.LAYOUTS,
  ...LabJack.Task.LAYOUTS,
  ...Modbus.Task.LAYOUTS,
  ...NI.Task.LAYOUTS,
  ...OPC.Task.LAYOUTS,
  [SELECTOR_LAYOUT_TYPE]: Selector,
  ...Sequence.LAYOUTS,
};

export const NAV_DRAWER_ITEMS: Layout.NavDrawerItem[] = [TOOLBAR_NAV_DRAWER_ITEM];
