// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { Kafka } from "@/feature/kafka";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { MQTT } from "@/feature/mqtt";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { PagerDuty } from "@/feature/pagerduty";
import { Panel } from "@/platform/panel";
import { Selector as Base } from "@/platform/selector";

const GROUPS: Base.Group[] = [
  {
    key: "ethercat",
    title: "EtherCAT",
    icon: <Icon.Logo.EtherCAT />,
    selectables: EtherCAT.Task.SELECTABLES,
  },
  {
    key: "http",
    title: "HTTP",
    icon: <Icon.Logo.HTTP />,
    selectables: HTTP.Task.SELECTABLES,
  },
  {
    key: "kafka",
    title: "Kafka",
    icon: <Icon.Logo.Kafka />,
    selectables: Kafka.Task.SELECTABLES,
  },
  {
    key: "labjack",
    title: "LabJack",
    icon: <Icon.Logo.LabJack />,
    selectables: LabJack.Task.SELECTABLES,
  },
  {
    key: "modbus",
    title: "Modbus",
    icon: <Icon.Logo.Modbus />,
    selectables: Modbus.Task.SELECTABLES,
  },
  {
    key: "mqtt",
    title: "MQTT",
    icon: <Icon.Logo.MQTT />,
    selectables: MQTT.Task.SELECTABLES,
  },
  { key: "ni", title: "NI", icon: <Icon.Logo.NI />, selectables: NI.Task.SELECTABLES },
  {
    key: "opcua",
    title: "OPC UA",
    icon: <Icon.Logo.OPCUA />,
    selectables: OPCUA.Task.SELECTABLES,
  },
  {
    key: "pagerduty",
    title: "PagerDuty",
    icon: <Icon.Logo.PagerDuty />,
    selectables: PagerDuty.Task.SELECTABLES,
  },
].filter((g) => g.selectables.length > 0);

const useCreateGranted = (): boolean => Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);

/** The task types, one group per integration, for a user who may create tasks. */
export const SECTION: Base.Section = {
  key: "tasks",
  groups: GROUPS,
  useVisible: useCreateGranted,
};

export const Selector = Base.create({
  sections: [SECTION],
  icon: <Icon.Task />,
  tabTitle: "Create task",
  text: "Create task",
  searchPlaceholder: "Search tasks",
});

export const SELECTOR_TAB_TYPE = "taskSelector";

export const useOpenSelector = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  return () => openTab({ variant: "view", type: SELECTOR_TAB_TYPE });
};
