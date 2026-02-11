// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";

import { EtherCAT } from "@/hardware/ethercat";
import { LabJack } from "@/hardware/labjack";
import { Modbus } from "@/hardware/modbus";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";

const PREFIXES = [
  EtherCAT.Task.PREFIX,
  LabJack.Task.PREFIX,
  Modbus.Task.PREFIX,
  NI.Task.PREFIX,
  OPC.Task.PREFIX,
] as const;
type Prefix = (typeof PREFIXES)[number];

const ICONS: Record<Prefix, Icon.ReactElement> = {
  [EtherCAT.Task.PREFIX]: <Icon.Device />,
  [LabJack.Task.PREFIX]: <Icon.Logo.LabJack />,
  [Modbus.Task.PREFIX]: <Icon.Logo.Modbus />,
  [NI.Task.PREFIX]: <Icon.Logo.NI />,
  [OPC.Task.PREFIX]: <Icon.Logo.OPC />,
};

export const getIcon = (type: string): Icon.ReactElement => {
  for (const prefix of PREFIXES) if (type.startsWith(prefix)) return ICONS[prefix];
  return <Icon.Task />;
};

const PREFIX_NAMES: Record<Prefix, string> = {
  [EtherCAT.Task.PREFIX]: "EtherCAT",
  [LabJack.Task.PREFIX]: "LabJack",
  [Modbus.Task.PREFIX]: "Modbus",
  [NI.Task.PREFIX]: "NI",
  [OPC.Task.PREFIX]: "OPC UA",
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
  for (let i = 1; i < words.length; i++) words[i] = caseconv.capitalize(words[i]);
  return `${words.join(" ")} Task`;
};
