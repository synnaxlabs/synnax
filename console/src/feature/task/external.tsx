// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";

import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPC } from "@/feature/opc";
import { PagerDuty } from "@/feature/pagerduty";
import { Selector, SELECTOR_TAB_TYPE } from "@/feature/task/Selector";
import { retrieveAndOpenTab } from "@/feature/task/tabs";
import { getIcon, parseType } from "@/feature/task/types";
import { type Command } from "@/platform/command";
import { type Import } from "@/platform/import";
import { type Panel } from "@/platform/panel";
import { type Range } from "@/platform/range";
import { type Task } from "@/platform/task";

export * from "@/feature/task/link";
export * from "@/feature/task/notifications";
export * from "@/feature/task/search";
export * from "@/feature/task/Selector";
export * from "@/feature/task/tabs";
export * from "@/feature/task/Toolbar";
export * from "@/feature/task/tree";
export * from "@/feature/task/types";
export * from "@/platform/task/external";

export const REGISTRY: Task.Registry = { getIcon, parseType };

export const COMMANDS: Command.Command[] = [
  ...EtherCAT.Task.COMMANDS,
  ...HTTP.Task.COMMANDS,
  ...LabJack.Task.COMMANDS,
  ...Modbus.Task.COMMANDS,
  ...NI.Task.COMMANDS,
  ...OPC.Task.COMMANDS,
  ...PagerDuty.Task.COMMANDS,
];

export const FILE_INGESTERS: Import.FileIngesters = {
  ...EtherCAT.Task.FILE_INGESTERS,
  ...HTTP.Task.FILE_INGESTERS,
  ...LabJack.Task.FILE_INGESTERS,
  ...Modbus.Task.FILE_INGESTERS,
  ...NI.Task.FILE_INGESTERS,
  ...OPC.Task.FILE_INGESTERS,
  ...PagerDuty.Task.FILE_INGESTERS,
};

export const TABS: Panel.Tabs = {
  [SELECTOR_TAB_TYPE]: Selector,
  ...EtherCAT.Task.TABS,
  ...HTTP.Task.TABS,
  ...LabJack.Task.TABS,
  ...Modbus.Task.TABS,
  ...NI.Task.TABS,
  ...OPC.Task.TABS,
  ...PagerDuty.Task.TABS,
};

export const SNAPSHOT_SERVICES: Range.SnapshotServices = {
  task: {
    icon: <Icon.Task />,
    onClick: async ({ id: { key } }, { client, openTab }) =>
      await retrieveAndOpenTab(client, key, openTab),
    onDelete: async ({ id: { key } }, { client }) => {
      if (client == null) throw new DisconnectedError();
      await client.tasks.delete(key);
    },
  },
};
