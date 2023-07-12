// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TooltipConfig } from "./TooltipConfig";

import { Tooltip as CoreTooltip, TooltipWrap } from "@/core/std/Tooltip/Tooltip";

type CoreTooltipType = typeof CoreTooltip;

interface TooltipType extends CoreTooltipType {
  Config: typeof TooltipConfig;
  wrap: typeof TooltipWrap;
}

export const Tooltip = CoreTooltip as TooltipType;

Tooltip.Config = TooltipConfig;
Tooltip.wrap = TooltipWrap;
