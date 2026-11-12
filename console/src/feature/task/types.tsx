// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { caseconv } from "@synnaxlabs/x";

import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { Kafka } from "@/feature/kafka";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { MQTT } from "@/feature/mqtt";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { PagerDuty } from "@/feature/pagerduty";
import { type Task } from "@/platform/task";

export const INTEGRATIONS: Task.Integration[] = [
  EtherCAT.Task.INTEGRATION,
  HTTP.Task.INTEGRATION,
  Kafka.Task.INTEGRATION,
  LabJack.Task.INTEGRATION,
  Modbus.Task.INTEGRATION,
  MQTT.Task.INTEGRATION,
  NI.Task.INTEGRATION,
  OPCUA.Task.INTEGRATION,
  PagerDuty.Task.INTEGRATION,
];

/** The read task types that a plot can show, by type. */
export const PLOT_CHANNELS: Record<string, Task.PlotChannels> = Object.fromEntries(
  INTEGRATIONS.flatMap(({ plotChannels }) => Object.entries(plotChannels ?? {})),
);

const TASK_ICON = <Icon.Task />;

export const getIcon = (type: string): Icon.ReactElement =>
  INTEGRATIONS.find(({ prefix }) => type.startsWith(prefix))?.icon ?? TASK_ICON;

export const parseType = (type: string): string => {
  const [first, ...rest] = type.split("_");
  const name =
    INTEGRATIONS.find(({ prefix }) => prefix === first)?.name ??
    caseconv.capitalize(first);
  return `${[name, ...rest].join(" ")} task`;
};
