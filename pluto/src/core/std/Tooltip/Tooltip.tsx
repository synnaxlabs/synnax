// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ComponentPropsWithoutRef,
  EventHandler,
  FC,
  MouseEvent,
  ReactElement,
  ReactNode,
  cloneElement,
  forwardRef,
  useRef,
  useState,
} from "react";

import { Box, CrudeOuterLocation, CrudeTimeSpan, TimeSpan, XY } from "@synnaxlabs/x";

import { CSS } from "@/core/css";
import { useTooltipConfig } from "@/core/std/Tooltip/TooltipConfig";

import "@/core/std/Tooltip/Tooltip.css";

export interface TooltipProps
  extends Omit<ComponentPropsWithoutRef<"div">, "children"> {
  delay?: CrudeTimeSpan;
  location?: CrudeOuterLocation;
  hide?: boolean;
  children: [ReactNode, ReactElement];
}

interface State {
  location: CrudeOuterLocation;
  position: XY;
}

const SIZE_THRESHOLD = 200;

const LOCATION_PREFERENCES: CrudeOuterLocation[] = ["right", "left", "bottom", "top"];

const getLocation = (container: Box, window: Box): CrudeOuterLocation => {
  for (const location of LOCATION_PREFERENCES) {
    const distance = Math.abs(window.loc(location) - container.loc(location));
    if (distance > SIZE_THRESHOLD) return location;
  }
  return LOCATION_PREFERENCES[0];
};

export const Tooltip = ({
  delay,
  children,
  location: propsLocation,
  hide = false,
}: TooltipProps): ReactElement => {
  const config = useTooltipConfig();
  const parsedDelay = new TimeSpan(delay ?? config.delay);
  const [state, setState] = useState<State | null>(null);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleVisibleChange = (e: MouseEvent, visible: boolean): void => {
    if (!visible || hide) return setState(null);
    config.startAccelerating();
    const conainer = new Box(e.target as HTMLElement);
    const window = new Box(document.documentElement);
    const location = propsLocation ?? getLocation(conainer, window);
    setState({
      location,
      position: conainer.locPoint(location),
    });
  };

  if (hide && state != null) setState(null);

  const handleMouseEnter: EventHandler<MouseEvent> = (e): void => {
    timeoutRef.current = setTimeout(
      () => handleVisibleChange(e, true),
      parsedDelay.milliseconds
    );
  };

  const handleMouseLeave: EventHandler<MouseEvent> = (e): void => {
    if (timeoutRef.current != null) clearInterval(timeoutRef.current);
    handleVisibleChange(e, false);
  };

  const [tip, children_] = children;

  return (
    <>
      {state != null && (
        <div
          className={CSS(CSS.B("tooltip"), CSS.loc(state.location))}
          style={{ ...state.position.css }}
        >
          {tip}
        </div>
      )}
      {cloneElement(children_, {
        onMouseEnter: handleMouseEnter,
        onMouseLeave: handleMouseLeave,
      })}
    </>
  );
};

export interface WrapTooltipProps {
  tooltipDelay?: TooltipProps["delay"];
  tooltip?: TooltipProps["children"][0];
  tooltipLocation?: TooltipProps["location"];
  hideTooltip?: TooltipProps["hide"];
}

export const TooltipWrap = <P extends {} = {}, E extends HTMLElement = HTMLElement>(
  Component: FC<P>
): ReturnType<typeof forwardRef<E, P & WrapTooltipProps>> => {
  const C = forwardRef<E, P & WrapTooltipProps>(
    ({ tooltipDelay, tooltip, tooltipLocation, ...props }, ref): ReactElement => {
      const c = <Component ref={ref} {...(props as P)} />;
      if (tooltip == null) return c;
      return (
        <Tooltip delay={tooltipDelay} location={tooltipLocation}>
          {tooltip}
          {c}
        </Tooltip>
      );
    }
  );
  C.displayName = `Tooltip.Wrap(${Component.displayName ?? Component.name})`;
  return C;
};
