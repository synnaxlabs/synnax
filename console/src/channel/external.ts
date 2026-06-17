// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Channel, type Icon } from "@synnaxlabs/pluto";

import { Calculated } from "@/channel/Calculated";
import { CALCULATED_LAYOUT_TYPE } from "@/channel/calculatedLayout";
import { Create, CREATE_LAYOUT_TYPE } from "@/channel/Create";
import { type Modals } from "@/modals";

export * from "@/channel/Calculated";
export * from "@/channel/calculatedLayout";
export * from "@/channel/Create";

export const MODALS: Record<string, Modals.Renderer> = {
  [CREATE_LAYOUT_TYPE]: Create,
  [CALCULATED_LAYOUT_TYPE]: Calculated,
};

export const ICONS: Record<string, Icon.FC> = {
  [CREATE_LAYOUT_TYPE]: Channel.CreateIcon,
  [CALCULATED_LAYOUT_TYPE]: Channel.CreateCalculatedIcon
};
