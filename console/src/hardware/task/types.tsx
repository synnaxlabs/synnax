// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type Icon as PIcon } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";

import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";

const PREFIXES = [LabJack.Task.PREFIX, NI.Task.PREFIX, OPC.Task.PREFIX] as const;
type Prefix = (typeof PREFIXES)[number];

const ICONS: Record<Prefix, PIcon.Element> = {
  [LabJack.Task.PREFIX]: <Icon.Logo.LabJack />,
  [NI.Task.PREFIX]: <Icon.Logo.NI />,
  [OPC.Task.PREFIX]: <Icon.Logo.OPC />,
};

export const getIcon = (type: string): PIcon.Element => {
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
  let prefixModified = false;
  for (const prefix of PREFIXES)
    if (words[0] === prefix) {
      prefixModified = true;
      words[0] = PREFIX_NAMES[prefix];
      break;
    }
  if (!prefixModified) words[0] = caseconv.capitalize(words[0]);
  for (let i = 1; i < words.length; i++) words[i] = caseconv.capitalize(words[i]);
  return `${words.join(" ")} Task`;
};
