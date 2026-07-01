// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/component/layout";
import { type Nav } from "@/component/nav";
import { type Palette } from "@/component/palette";
import { type Export } from "@/service/export";
import { type Import } from "@/service/import";
import { EtherCAT } from "@/service/ethercat";
import { HTTP } from "@/service/http";
import { LabJack } from "@/service/labjack";
import { Modbus } from "@/service/modbus";
import { NI } from "@/service/ni";
import { OPC } from "@/service/opc";
import { PagerDuty } from "@/service/pagerduty";
import { Selector, SELECTOR_LAYOUT_TYPE } from "@/service/task/Selector";
import { TOOLBAR } from "@/service/task/Toolbar";

export * from "@/service/task/createIngester";
export * from "@/service/task/export";
export * from "@/service/task/Form";
export * from "@/service/task/layouts";
export * from "@/service/task/link";
export * from "@/service/task/ontology";
export * from "@/service/task/Selector";
export * from "@/service/task/Toolbar";
export * from "@/service/task/UtilityButtons";

export const COMMANDS: Palette.Command[] = [
  ...EtherCAT.Task.COMMANDS,
  ...HTTP.Task.COMMANDS,
  ...LabJack.Task.COMMANDS,
  ...Modbus.Task.COMMANDS,
  ...NI.Task.COMMANDS,
  ...OPC.Task.COMMANDS,
  ...PagerDuty.Task.COMMANDS,
];

export const EXTRACTORS: Export.Extractors = {
  ...EtherCAT.Task.EXTRACTORS,
  ...HTTP.Task.EXTRACTORS,
  ...LabJack.Task.EXTRACTORS,
  ...Modbus.Task.EXTRACTORS,
  ...NI.Task.EXTRACTORS,
  ...OPC.Task.EXTRACTORS,
  ...PagerDuty.Task.EXTRACTORS,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  ...EtherCAT.Task.FILE_INGESTERS,
  ...HTTP.Task.FILE_INGESTERS,
  ...LabJack.Task.FILE_INGESTERS,
  ...Modbus.Task.FILE_INGESTERS,
  ...NI.Task.FILE_INGESTERS,
  ...OPC.Task.FILE_INGESTERS,
  ...PagerDuty.Task.FILE_INGESTERS,
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  ...EtherCAT.Task.LAYOUTS,
  ...HTTP.Task.LAYOUTS,
  ...LabJack.Task.LAYOUTS,
  ...Modbus.Task.LAYOUTS,
  ...NI.Task.LAYOUTS,
  ...OPC.Task.LAYOUTS,
  ...PagerDuty.Task.LAYOUTS,
  [SELECTOR_LAYOUT_TYPE]: Selector,
};

export const NAV_DRAWER_ITEMS: Nav.Item[] = [TOOLBAR];
