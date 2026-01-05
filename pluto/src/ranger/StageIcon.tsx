// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeRange } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Icon } from "@/icon";
import { getStage, STAGE_ICONS } from "@/ranger/stage";

export interface StageIconProps extends Icon.IconProps {
  timeRange: CrudeTimeRange;
}

export const StageIcon = ({ timeRange, ...rest }: StageIconProps): ReactElement => {
  const I = STAGE_ICONS[getStage(timeRange)];
  return <I {...rest} />;
};
