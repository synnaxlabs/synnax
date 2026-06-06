// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/polygon/config";
import { Theming } from "@/theming";

export const DEFAULT_POLYGON_SIDE_LENGTH = 20;

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
}

const calculatePolygonVertices = (
  numSides: number,
  sideLength: number,
  rotationDeg: number = 0,
  padding: number = 4,
): Array<{ x: number; y: number }> => {
  const angleStep = (2 * Math.PI) / numSides;
  const rotationRad = (rotationDeg * Math.PI) / 180;
  const radius = sideLength / (2 * Math.sin(Math.PI / numSides));
  const center = radius + padding / 2;

  return Array.from({ length: numSides }).map((_, i) => {
    const angle = angleStep * i + rotationRad - Math.PI / 2;
    return {
      x: center + radius * Math.cos(angle),
      y: center + radius * Math.sin(angle),
    };
  });
};

const generateRoundedPolygonPath = (
  vertices: { x: number; y: number }[],
  sideLength: number,
  cornerRounding: number,
): string => {
  const path: string[] = [];
  const r = Math.min(cornerRounding, sideLength / 2);

  const len = vertices.length;
  if (len < 3 || r <= 0)
    return `M ${vertices.map((v) => `${v.x},${v.y}`).join(" L ")} Z`;

  for (let i = 0; i < len; i++) {
    const prev = vertices[(i - 1 + len) % len];
    const curr = vertices[i];
    const next = vertices[(i + 1) % len];

    const dx1 = curr.x - prev.x;
    const dy1 = curr.y - prev.y;
    const dx2 = next.x - curr.x;
    const dy2 = next.y - curr.y;

    const len1 = Math.hypot(dx1, dy1);
    const len2 = Math.hypot(dx2, dy2);

    const offset1 = r / len1;
    const offset2 = r / len2;

    const p1x = curr.x - dx1 * offset1;
    const p1y = curr.y - dy1 * offset1;
    const p2x = curr.x + dx2 * offset2;
    const p2y = curr.y + dy2 * offset2;

    if (i === 0) path.push(`M ${p1x},${p1y}`);
    else path.push(`L ${p1x},${p1y}`);

    path.push(`Q ${curr.x},${curr.y} ${p2x},${p2y}`);
  }

  path.push("Z");
  return path.join(" ");
};

export const Polygon = ({
  numSides,
  sideLength,
  rotation = 0,
  color: colorVal,
  backgroundColor,
  className,
  cornerRounding,
  strokeWidth,
}: RenderProps): ReactElement => {
  const theme = Theming.use();
  const padding = useMemo(() => 2 * ((strokeWidth ?? 2) + 1), [strokeWidth]);
  const vertices = useMemo(
    () => calculatePolygonVertices(numSides, sideLength, rotation, padding),
    [numSides, sideLength, rotation, padding],
  );
  const path = useMemo(
    () => generateRoundedPolygonPath(vertices, sideLength, cornerRounding ?? 0),
    [vertices, sideLength, cornerRounding],
  );
  const size = useMemo(
    () => 2 * (sideLength / (2 * Math.sin(Math.PI / numSides))) + padding,
    [sideLength, numSides, padding],
  );
  return (
    <Primitive.Div className={CSS(className, CSS.B("polygon"))}>
      <Primitive.SVG dimensions={{ width: size, height: size }} color={colorVal}>
        <Primitive.Path
          d={path}
          fill={color.cssString(backgroundColor ?? theme.colors.gray.l1)}
          strokeWidth={strokeWidth ?? 2}
        />
      </Primitive.SVG>
    </Primitive.Div>
  );
};
