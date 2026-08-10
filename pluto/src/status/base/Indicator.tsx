// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { type ReactElement } from "react";

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
      cx="6.5"
      cy="6.5"
      r="5.5"
      fill="none"
      stroke="currentColor"
      strokeWidth="1"
      opacity="0.45"
    />
    <circle cx="6.5" cy="6.5" r="3" />
  </svg>
);

const Concentric = Icon.wrapSVGIcon(ConcentricSVG, "status-concentric");

export interface IndicatorProps extends Icon.IconProps {
  variant?: status.Variant;
}

export const Indicator = ({ variant, ...rest }: IndicatorProps): ReactElement =>
  variant === "loading" ? (
    <Icon.Loading {...rest} />
  ) : (
    <Concentric
      color={variant != null ? VARIANT_COLORS[variant] : undefined}
      {...rest}
    />
  );
