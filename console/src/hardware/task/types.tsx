// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { caseconv } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";

export const PREFIXES = [LabJack.Task.PREFIX, NI.Task.PREFIX, OPC.Task.PREFIX] as const;
export type Prefix = (typeof PREFIXES)[number];

export const ICONS: Record<Prefix, ReactElement> = {
  [LabJack.Task.PREFIX]: <Icon.Logo.LabJack />,
  [NI.Task.PREFIX]: <Icon.Logo.NI />,
  [OPC.Task.PREFIX]: <Icon.Logo.OPC />,
};

export const getIcon = (type: string): ReactElement => {
  for (const prefix of PREFIXES) if (type.startsWith(prefix)) return ICONS[prefix];
  return <Icon.Task />;
};

const PREFIX_NAMES: Record<Prefix, string> = {
  [LabJack.Task.PREFIX]: "LabJack",
  [NI.Task.PREFIX]: "NI",
  [OPC.Task.PREFIX]: "OPC UA",
};

export const parseType = (type: string): string => {
  const words = type.split("_");
  for (const prefix of PREFIXES)
    if (words[0] === prefix) {
      words[0] = PREFIX_NAMES[prefix];
      break;
    }
  for (let i = 1; i < words.length; i++) words[i] = caseconv.capitalize(words[i]);
  return `${words.join(" ")} Task`;
};
