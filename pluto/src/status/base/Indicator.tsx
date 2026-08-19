// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { cloneElement, type ReactElement } from "react";

import { CSS } from "@/css";
import { Icon } from "@/icon";
import { VARIANT_COLORS } from "@/status/base/colors";

/**
 * A faint outer track around a solid core, drawn on a 13 unit grid so it lands on whole
 * pixels wherever the font size is 13px. The colour arrives through `style` because
 * `.pluto-icon` sets `color: inherit`, and a class rule outranks a presentation
 * attribute.
 */
const ConcentricSVG: Icon.SVGFC = ({ color, style, ...rest }) => (
  <svg
    viewBox="0 0 13 13"
    fill="currentColor"
    style={color != null ? { ...style, color } : style}
    {...rest}
  >
    <circle
      className={CSS.BE("status-indicator", "track")}
      cx="6.5"
      cy="6.5"
      r="5.5"
      fill="none"
      stroke="currentColor"
      strokeWidth="1"
      opacity="0.45"
    />
    <circle className={CSS.BE("status-indicator", "core")} cx="6.5" cy="6.5" r="3" />
  </svg>
);

const Concentric = Icon.wrapSVGIcon(ConcentricSVG, "status-concentric");

/** Props for {@link Indicator}. */
export interface IndicatorProps extends Icon.IconProps {
  variant?: status.Variant;
  /** Replaces the concentric glyph, tinted with the variant color. */
  children?: ReactElement<Icon.IconProps>;
}

/**
 * The dot that carries a status variant's color. A loading variant spins instead, and
 * a child icon takes the color in place of the dot.
 */
export const Indicator = ({
  variant,
  children,
  ...rest
}: IndicatorProps): ReactElement => {
  if (variant === "loading") return <Icon.Loading {...rest} />;
  const color = variant != null ? VARIANT_COLORS[variant] : undefined;
  if (children == null) return <Concentric color={color} {...rest} />;
  const { className, style, ...others } = rest;
  // The color rides on style because `.pluto-icon` sets `color: inherit`, and a
  // class rule outranks a presentation attribute.
  return cloneElement(children, {
    ...others,
    className: CSS.cls(children.props.className, className),
    style: { ...children.props.style, ...(color != null && { color }), ...style },
  });
};
