// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, type ReactElement } from "react";

import { Dialog, type DialogProps } from "@/tooltip/Dialog";

export interface WrapProps {
  tooltipDelay?: DialogProps["delay"];
  tooltip?: DialogProps["children"][0];
  tooltipLocation?: DialogProps["location"];
  hideTooltip?: DialogProps["hide"];
}

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
export const wrap = <P extends {} = {}>(Component: FC<P>): FC<P & WrapProps> => {
  const C = ({
    tooltipDelay,
    tooltip,
    tooltipLocation,
    ...rest
  }: P & WrapProps): ReactElement => {
    const c = <Component {...(rest as unknown as P)} />;
    if (tooltip == null) return c;
    return (
      <Dialog delay={tooltipDelay} location={tooltipLocation}>
        {tooltip}
        {c}
      </Dialog>
    );
  };
  C.displayName = `Tooltip.Wrap(${Component.displayName ?? Component.name})`;
  return C;
};
