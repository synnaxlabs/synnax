// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CSSProperties, type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/gauge/config";
import { Text } from "@/text";

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
}

const CONTAINER_STYLE: CSSProperties = {
  width: 67,
  height: 67,
  position: "relative",
};

const SVG_STYLE: CSSProperties = { position: "absolute" };

const METRICS_STYLE: CSSProperties = {
  position: "absolute",
  top: "60%",
  left: "50%",
  transform: "translate(-50%, -50%)",
  textAlign: "center",
};

export const Gauge = ({ color: c, className }: RenderProps): ReactElement => {
  const radius = 27;
  const strokeWidth = 5;
  const centerX = 33.5;
  const centerY = 33.5;

  const startAngle = 135 * (Math.PI / 180);
  const endAngle = 45 * (Math.PI / 180);
  const valueAngle = 135 + 270 * 0.5;
  const valueEndAngle = valueAngle * (Math.PI / 180);

  const backgroundPath = `
    M ${centerX + radius * Math.cos(startAngle)} ${centerY + radius * Math.sin(startAngle)}
    A ${radius} ${radius} 0 1 1 ${centerX + radius * Math.cos(endAngle)} ${centerY + radius * Math.sin(endAngle)}
  `;

  const valuePath = `
    M ${centerX + radius * Math.cos(startAngle)} ${centerY + radius * Math.sin(startAngle)}
    A ${radius} ${radius} 0 ${valueAngle - 135 > 180 ? 1 : 0} 1 ${centerX + radius * Math.cos(valueEndAngle)} ${centerY + radius * Math.sin(valueEndAngle)}
  `;

  const style = useMemo<CSSProperties>(
    () => ({
      ...CONTAINER_STYLE,
      [CSS.var("symbol-color")]: Primitive.symbolColorVar(c),
    }),
    [c],
  );

  return (
    <div className={CSS(CSS.B("symbol-colored"), className)} style={style}>
      <svg width="67" height="67" style={SVG_STYLE}>
        <path
          d={backgroundPath}
          fill="none"
          stroke="var(--pluto-gray-l5)"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
        <path
          d={valuePath}
          fill="none"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
      </svg>
      <div style={METRICS_STYLE}>
        <Text.Text level="h5" weight="bold">
          750
        </Text.Text>
        <Text.Text level="small" color={7}>
          RPM
        </Text.Text>
      </div>
    </div>
  );
};
