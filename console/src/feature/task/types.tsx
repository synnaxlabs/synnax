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

import { CAN } from "@/feature/can";
import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { PagerDuty } from "@/feature/pagerduty";
import { Serial } from "@/feature/serial";
import { TCP } from "@/feature/tcp";
import { UDP } from "@/feature/udp";

const PREFIXES = [
  CAN.Task.PREFIX,
  EtherCAT.Task.PREFIX,
  HTTP.Task.PREFIX,
  LabJack.Task.PREFIX,
  Modbus.Task.PREFIX,
  NI.Task.PREFIX,
  OPCUA.Task.PREFIX,
  PagerDuty.Task.PREFIX,
  Serial.Task.PREFIX,
  TCP.Task.PREFIX,
  UDP.Task.PREFIX,
] as const;
type Prefix = (typeof PREFIXES)[number];

const ICONS: Record<Prefix, Icon.ReactElement> = {
  [CAN.Task.PREFIX]: <Icon.Hardware />,
  [Serial.Task.PREFIX]: <Icon.Connect />,
  [TCP.Task.PREFIX]: <Icon.Link />,
  [UDP.Task.PREFIX]: <Icon.Bridge />,
  [EtherCAT.Task.PREFIX]: <Icon.Logo.EtherCAT />,
  [HTTP.Task.PREFIX]: <Icon.Logo.HTTP />,
  [LabJack.Task.PREFIX]: <Icon.Logo.LabJack />,
  [Modbus.Task.PREFIX]: <Icon.Logo.Modbus />,
  [NI.Task.PREFIX]: <Icon.Logo.NI />,
  [OPCUA.Task.PREFIX]: <Icon.Logo.OPCUA />,
  [PagerDuty.Task.PREFIX]: <Icon.Logo.PagerDuty />,
};

export const getIcon = (type: string): Icon.ReactElement => {
  for (const prefix of PREFIXES) if (type.startsWith(prefix)) return ICONS[prefix];
  return <Icon.Task />;
};

const PREFIX_NAMES: Record<Prefix, string> = {
  [CAN.Task.PREFIX]: "CAN",
  [Serial.Task.PREFIX]: "Serial",
  [TCP.Task.PREFIX]: "TCP",
  [UDP.Task.PREFIX]: "UDP",
  [EtherCAT.Task.PREFIX]: "EtherCAT",
  [HTTP.Task.PREFIX]: "HTTP",
  [LabJack.Task.PREFIX]: "LabJack",
  [Modbus.Task.PREFIX]: "Modbus",
  [NI.Task.PREFIX]: "NI",
  [OPCUA.Task.PREFIX]: "OPC UA",
  [PagerDuty.Task.PREFIX]: "PagerDuty",
};

export const parseType = (type: string): string => {
  const words = type.split("_");
  let isFirstWordModified = false;
  for (const prefix of PREFIXES)
    if (words[0] === prefix) {
      isFirstWordModified = true;
      words[0] = PREFIX_NAMES[prefix];
      break;
    }
  if (!isFirstWordModified) words[0] = caseconv.capitalize(words[0]);
  return `${words.join(" ")} task`;
};
