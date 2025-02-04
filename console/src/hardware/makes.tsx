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
import { z } from "zod";

import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";

export const MAKES = [NI.MAKE, LabJack.MAKE, OPC.MAKE] as const;
export const makeZ = z.enum(MAKES);
export type Make = z.infer<typeof makeZ>;

export const MAKE_ICONS: Record<Make, PIcon.Element> = {
  [LabJack.MAKE]: <Icon.Logo.LabJack />,
  [NI.MAKE]: <Icon.Logo.NI />,
  [OPC.MAKE]: <Icon.Logo.OPC />,
};
