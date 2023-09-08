// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ComponentPropsWithoutRef,
  type EventHandler,
  type MouseEvent,
  type ReactElement,
  type ReactNode,
  cloneElement,
  useRef,
  useState,
} from "react";

import {
  Box,
  type CrudeLocation,
  type CrudeOuterLocation,
  type CrudeTimeSpan,
  type CrudeXLocation,
  type CrudeXYLocation,
  type CrudeYLocation,
  type Dimensions,
  Location,
  TimeSpan,
  type XY,
  XYLocation,
} from "@synnaxlabs/x";
import { createPortal } from "react-dom";

import { CSS } from "@/css";
import { useConfig } from "@/tooltip/Config";

import "@/tooltip/Dialog.css";

export interface DialogProps extends Omit<ComponentPropsWithoutRef<"div">, "children"> {
  delay?: CrudeTimeSpan;
  location?: CrudeOuterLocation | Partial<CrudeXYLocation>;
  hide?: boolean;
  children: [ReactNode, ReactElement];
}

interface State {
  location: XYLocation;
  position: XY;
  elDims: Dimensions;
}

const SIZE_THRESHOLD = 150;

const Y_LOCATION_PREFERENCES: CrudeYLocation[] = ["top", "bottom"];
const X_LOCATION_PREFERENCES: CrudeXLocation[] = ["left", "right"];
const OUTER_LOCATION_PREFERENCES: CrudeOuterLocation[] = [
  ...X_LOCATION_PREFERENCES,
  ...Y_LOCATION_PREFERENCES,
];
const LOCATION_PREFERENCES: CrudeLocation[] = [...OUTER_LOCATION_PREFERENCES, "center"];

const LOCATION_TRANSLATIONS: Record<string, (xy: XY, container: Box) => XY> = {
  [XYLocation.TOP_RIGHT.toString()]: (xy, c) => xy.translateX(-c.width),
  [XYLocation.TOP_LEFT.toString()]: (xy, c) => xy.translateX(c.width),
  [XYLocation.BOTTOM_RIGHT.toString()]: (xy, c) => xy.translateX(-c.width),
  [XYLocation.BOTTOM_LEFT.toString()]: (xy, c) => xy.translateX(c.width),
};

const bestLocation = <C extends CrudeLocation>(
  container: Box,
  window: Box,
  options: C[],
): C => {
  for (const location of options) {
    const distance = Math.abs(window.loc(location) - container.loc(location));
    if (distance > SIZE_THRESHOLD) return location;
  }
  return options[0];
};

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
export const Dialog = ({
  delay,
  children,
  location: cornerOrLocation,
  hide = false,
}: DialogProps): ReactElement => {
  const config = useConfig();
  const parsedDelay = new TimeSpan(delay ?? config.delay);
  const [state, setState] = useState<State | null>(null);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleVisibleChange = (e: MouseEvent, visible: boolean): void => {
    if (!visible || hide) return setState(null);
    config.startAccelerating();
    const container = new Box(e.target as HTMLElement);
    const window = new Box(document.documentElement);
    const parse = Location.looseZ.safeParse(cornerOrLocation);

    const chooseRemainingLocation = (first: Location): Location => {
      let preferences: CrudeLocation[];
      if (first.isCenter) {
        preferences = OUTER_LOCATION_PREFERENCES;
      } else if (first.isX) preferences = ["center", ...Y_LOCATION_PREFERENCES];
      else preferences = ["center", ...X_LOCATION_PREFERENCES];
      return new Location(bestLocation(container, window, preferences));
    };

    let xy: XYLocation = XYLocation.CENTER;
    if (parse.success) {
      xy = new XYLocation(parse.data, chooseRemainingLocation(parse.data));
    } else if (cornerOrLocation != null) {
      const v = cornerOrLocation as Partial<CrudeXYLocation>;
      if (v.x == null && v.y != null)
        v.x = chooseRemainingLocation(new Location(v.y)).crude as CrudeXLocation;
      else if (v.y == null && v.x != null)
        v.y = chooseRemainingLocation(new Location(v.x)).crude as CrudeYLocation;
      else if (v.x == null && v.y == null) {
        v.x = new Location(bestLocation(container, window, LOCATION_PREFERENCES))
          .crude as CrudeXLocation;
        v.y = chooseRemainingLocation(new Location(v.x)).crude as CrudeYLocation;
      }
      xy = new XYLocation(v as CrudeXYLocation);
    } else {
      const chosen = new Location(
        bestLocation(container, window, LOCATION_PREFERENCES),
      );
      xy = new XYLocation(chosen, chooseRemainingLocation(chosen));
    }

    let pos = container.xyLoc(xy);
    const translate = LOCATION_TRANSLATIONS[xy.toString()];
    if (translate != null) pos = translate(pos, container);

    setState({
      location: xy,
      position: pos,
      elDims: container.dims,
    });
  };

  if (hide && state != null) setState(null);

  const handleMouseEnter: EventHandler<MouseEvent> = (e): void => {
    timeoutRef.current = setTimeout(
      () => handleVisibleChange(e, true),
      parsedDelay.milliseconds,
    );
  };

  const handleMouseLeave: EventHandler<MouseEvent> = (e): void => {
    if (timeoutRef.current != null) clearInterval(timeoutRef.current);
    handleVisibleChange(e, false);
  };

  const [tip, children_] = children;

  return (
    <>
      {state != null &&
        createPortal(
          <div
            className={CSS(
              CSS.B("tooltip"),
              CSS.loc(state.location.x),
              CSS.loc(state.location.y),
            )}
            style={{
              ...state.position.css,
              // @ts-expect-error
              "--el-width": CSS.px(state.elDims.width),
            }}
          >
            {tip}
          </div>,
          document.body,
        )}
      {cloneElement(children_, {
        onMouseEnter: handleMouseEnter,
        onMouseLeave: handleMouseLeave,
      })}
    </>
  );
};
