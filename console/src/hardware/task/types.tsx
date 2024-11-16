// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type ReactElement } from "react";

import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";

export const TYPE_PREFIXES = [
  LabJack.Task.PREFIX,
  NI.Task.PREFIX,
  OPC.Task.PREFIX,
] as const;
export type TypePrefix = (typeof TYPE_PREFIXES)[number];

export const TASK_ICONS: Record<TypePrefix, ReactElement> = {
  [LabJack.Task.PREFIX]: <Icon.Logo.LabJack />,
  [NI.Task.PREFIX]: <Icon.Logo.NI />,
  [OPC.Task.PREFIX]: <Icon.Logo.OPC />,
};

export const getIcon = (type: string): ReactElement => {
  for (const prefix of TYPE_PREFIXES)
    if (type.startsWith(prefix)) return TASK_ICONS[prefix];
  return <Icon.Task />;
};
