// Copyright 2025 Synnax Labs, Inc.
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
import { Sequence } from "@/hardware/task/sequence";

const PREFIXES = [
  LabJack.Task.PREFIX,
  NI.Task.PREFIX,
  OPC.Task.PREFIX,
  Sequence.TYPE,
] as const;
type Prefix = (typeof PREFIXES)[number];

const ICONS: Record<Prefix, PIcon.Element> = {
  [LabJack.Task.PREFIX]: <Icon.Logo.LabJack />,
  [NI.Task.PREFIX]: <Icon.Logo.NI />,
  [OPC.Task.PREFIX]: <Icon.Logo.OPC />,
  [Sequence.TYPE]: <Icon.Control />,
};

export const getIcon = (type: string): PIcon.Element => {
  for (const prefix of PREFIXES) if (type.startsWith(prefix)) return ICONS[prefix];
  return <Icon.Task />;
};

const PREFIX_NAMES: Record<Prefix, string> = {
  [LabJack.Task.PREFIX]: "LabJack",
  [NI.Task.PREFIX]: "NI",
  [OPC.Task.PREFIX]: "OPC UA",
  [Sequence.TYPE]: "Sequence",
};

export const parseType = (type: string): string => {
  if (type === Sequence.TYPE) return "Control Sequence";
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
