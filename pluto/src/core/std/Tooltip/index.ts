// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tooltip as CoreTooltip, TooltipWrap } from "@/core/std/Tooltip/Tooltip";
import { TooltipConfig } from "@/core/std/Tooltip/TooltipConfig";

export type { TooltipConfigProps } from "@/core/std/Tooltip/TooltipConfig";
export type { TooltipProps } from "@/core/std/Tooltip/Tooltip";

type CoreTooltipType = typeof CoreTooltip;

interface TooltipType extends CoreTooltipType {
  /**
   * Sets the default configuration for all tooltips in its children.
   *
   * @param props - The props for the tooltip config.
   * @param props.delay - The delay before the tooltip appears, in milliseconds.
   * @default 500ms.
   * @param props.accelerate - Whether to enable accelerated visibility of tooltps for
   * a short period of time after the user has hovered over a first tooltip.
   * @default true.
   * @param props.acceleratedDelay - The delay before the tooltip appears when
   * accelerated visibility is enabled.
   * @default 100 ms.
   * @param props.acceleratedDuration - The duration of accelerated visibility.
   * @default 10 seconds.
   */
  Config: typeof TooltipConfig;
  /**
   * A higher order component that wraps a React component and attaches an optional,
   * prop-set tooltip to it.
   *
   * @param Component - Any React component.
   *
   * @returns The Wrapped React component, with the following props added:
   *
   * @param props.tooltip - An optional ReactNode containing the tooltip content. If
   * the prop is not specified, a tooltip is not shown.
   * @param props.tooltipDelay - The delay before the tooltip appears, in milliseconds. This prop
   * overrides the value set in Tooltip.Config. Defaults to the value set in Tooltip.Config,
   * which defaults to 500ms.
   * @param props.tooltipHide - Force the tooltip to remain hidden, even when the user hovers
   * over the element it is attached to.
   * @default false.
   * @param props.tooltipLocation - The location for the tooltip to appear relative to the
   * element it is attached to. If unspecified, the tooltip automatically chooses a
   * location based on the element's position on the screen.
   */
  wrap: typeof TooltipWrap;
}

/**
 * A tooltip that appears when the user hovers over an element.
 *
 * @param props - The props for the tooltip. Unlisted props are passed to the underlying
 * div element.
 * @param props.children - A ReactNode to render as the tooltip's content, followed by
 * a ReactElement to attach the tooltip to.
 * @param props.location - The location for the tooltip to appear relative to the
 * element it is attached to. If unspecified, the tooltip automatically chooses a
 * location based on the element's position on the screen.
 * @param props.hide - Force the tooltip to remain hidden, even when the user hovers
 * over the element it is attached to.
 * @default false.
 * @param props.delay - The delay before the tooltip appears, in milliseconds. This prop
 * overrides the value set in Tooltip.Config.
 * @default the value set in Tooltip.Config, which defaults to 500ms.
 */
export const Tooltip = CoreTooltip as TooltipType;

Tooltip.Config = TooltipConfig;
Tooltip.wrap = TooltipWrap;
