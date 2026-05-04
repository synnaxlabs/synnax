// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { BaseEdge, type BaseEdgeProps } from "@xyflow/react";
import { type ReactElement, useMemo } from "react";

export interface BaseProps extends Omit<BaseEdgeProps, "color"> {
  color: color.Color;
}

const INTERACTION_WIDTH = 30;

export const Base = ({
  style: baseStyle,
  color: stroke,
  ...props
}: BaseProps): ReactElement => {
  const style = useMemo(
    () => ({ ...baseStyle, stroke: color.cssString(stroke) }),
    [stroke, baseStyle],
  );
  return <BaseEdge {...props} interactionWidth={INTERACTION_WIDTH} style={style} />;
};
