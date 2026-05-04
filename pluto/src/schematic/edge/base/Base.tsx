import { color } from "@synnaxlabs/x";
import { BaseEdge, type BaseEdgeProps } from "@xyflow/react";
import { type ReactElement, useMemo } from "react";

export interface BaseProps extends Omit<BaseEdgeProps, "color"> {
  color: color.Color;
}

const INTERACTION_WIDTH = 30; // px

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
