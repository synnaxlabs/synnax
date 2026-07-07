// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPC } from "@/feature/opc";
import { PagerDuty } from "@/feature/pagerduty";
import { createLayout } from "@/feature/task/layouts";
import { Selector } from "@/feature/task/Selector";
import { TOOLBAR } from "@/feature/task/Toolbar";
import { getIcon, parseType } from "@/feature/task/types";
import { type Command } from "@/platform/command";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Layout } from "@/platform/layout";
import { type Nav } from "@/platform/nav";
import { Task } from "@/platform/task";

export * from "@/feature/task/layouts";
export * from "@/feature/task/link";
export * from "@/feature/task/useMosaicDrop";
export * from "@/feature/task/tree";
export * from "@/feature/task/search";
export * from "@/feature/task/Selector";
export * from "@/feature/task/Toolbar";
export * from "@/feature/task/types";
export * from "@/platform/task/external";

export const REGISTRY: Task.Registry = { createLayout, getIcon, parseType };

export const COMMANDS: Command.Command[] = [
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
  [Task.SELECTOR_LAYOUT_TYPE]: Selector,
};

export const NAV_DRAWER_ITEMS: Nav.Item[] = [TOOLBAR];
