// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/icon/Icon.css";

import { color, type location } from "@synnaxlabs/x";
import {
  type FC as ReactFC,
  type ReactElement as BaseReactElement,
  type Ref,
} from "react";
import { type IconBaseProps } from "react-icons";

import { CSS } from "@/css";
import { type Text } from "@/text";

export interface IconProps extends Omit<IconBaseProps, "color" | "children"> {
  ref?: Ref<SVGSVGElement>;
  color?: color.Crude | Text.Shade;
}

/**
 * A type representing a React element that renders an icon. This element encapsulates
 * the visual representation of an icon component, typically rendered as an SVG with the
 * specified base properties.
 */
export interface ReactElement extends BaseReactElement<IconProps> {}

const BASE_SIZE = 24;
const SUB_SIZE = 12;
const SUB_POSITIONS: Record<location.CornerXYString, { x: number; y: number }> = {
  topRight: { x: BASE_SIZE - SUB_SIZE, y: 0 },
  topLeft: { x: 0, y: 0 },
  bottomLeft: { x: 0, y: BASE_SIZE - SUB_SIZE },
  bottomRight: { x: BASE_SIZE - SUB_SIZE, y: BASE_SIZE - SUB_SIZE },
};

const createSubIcon = (
  key: location.CornerXYString,
  Icon: FC | undefined,
): ReactElement | null => {
  if (Icon == null) return null;
  return (
    <g transform={`translate(${SUB_POSITIONS[key].x}, ${SUB_POSITIONS[key].y})`}>
      <circle
        className={CSS.BE("sub", "bg")}
        r={SUB_SIZE / 2}
        cx={SUB_SIZE / 2}
        cy={SUB_SIZE / 2}
      />
      <Icon className={CSS(CSS.B("sub"), CSS.M(key))} size={SUB_SIZE} />
    </g>
  );
};

export interface FC extends ReactFC<IconProps> {}

interface WrapIconOpts {
  className?: string;
}

export interface SVGFC extends ReactFC<IconBaseProps> {}

const parseColor = (c?: color.Crude | Text.Shade): string | undefined => {
  if (typeof c === "number") return `var(--pluto-gray-l${c})`;
  return color.cssString(c);
};

export const wrapSVGIcon = (
  Base: SVGFC,
  name: string,
  { className }: WrapIconOpts = {},
): FC => {
  const typeClass = CSS.BM("icon", name);
  const O: FC = ({ className: pClassName, color: c, ...rest }) => {
    c = parseColor(c);
    return (
      <Base
        className={CSS(CSS.B("icon"), pClassName, className, typeClass)}
        aria-label={rest["aria-label"] ?? typeClass}
        color={c}
        {...rest}
      />
    );
  };
  O.displayName = Base.displayName || Base.name;
  return O;
};

export const createComposite = (
  Base: FC,
  { topRight, topLeft, bottomLeft, bottomRight }: Record<string, FC | undefined>,
): FC => {
  if (topRight == null && topLeft == null && bottomLeft == null && bottomRight == null)
    return Base;

  const topRightEl = createSubIcon("topRight", topRight);
  const topLeftEl = createSubIcon("topLeft", topLeft);
  const bottomLeftEl = createSubIcon("bottomLeft", bottomLeft);
  const bottomRightEl = createSubIcon("bottomRight", bottomRight);

  const Composite = ({ className, color: c, ...rest }: IconProps) => {
    c = parseColor(c);
    return (
      <svg
        className={CSS(CSS.B("icon"), CSS.BM("icon", "composite"))}
        viewBox="0 0 24 24"
        height="1em"
        width="1em"
        {...rest}
      >
        <Base className={className} {...rest} size={20} x={2} y={2} />
        {topRightEl}
        {topLeftEl}
        {bottomLeftEl}
        {bottomRightEl}
      </svg>
    );
  };
  Composite.displayName = Base.displayName || Base.name;
  return Composite;
};
