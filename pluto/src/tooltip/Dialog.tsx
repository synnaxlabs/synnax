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
  type dimensions,
  xy,
  location,
  type CrudeTimeSpan,
  box,
  TimeSpan,
} from "@synnaxlabs/x";
import { createPortal } from "react-dom";

import { CSS } from "@/css";
import { useConfig } from "@/tooltip/Config";

import "@/tooltip/Dialog.css";

export interface DialogProps extends Omit<ComponentPropsWithoutRef<"div">, "children"> {
  delay?: CrudeTimeSpan;
  location?: location.Outer | Partial<location.XY>;
  hide?: boolean;
  children: [ReactNode, ReactElement];
}

interface State {
  location: location.XY;
  position: xy.XY;
  elDims: dimensions.Dimensions;
}

const SIZE_THRESHOLD = 150;

const Y_LOCATION_PREFERENCES: location.Y[] = ["top", "bottom"];
const X_LOCATION_PREFERENCES: location.X[] = ["left", "right"];
const OUTER_LOCATION_PREFERENCES: location.Outer[] = [
  ...X_LOCATION_PREFERENCES,
  ...Y_LOCATION_PREFERENCES,
];
const LOCATION_PREFERENCES: location.Location[] = [
  ...OUTER_LOCATION_PREFERENCES,
  "center",
];

const LOCATION_TRANSLATIONS: Record<string, (p: xy.XY, container: box.Box) => xy.XY> = {
  [location.xyToString(location.TOP_RIGHT)]: (p, c) => xy.translateX(p, -box.width(c)),
  [location.xyToString(location.TOP_LEFT)]: (p, c) => xy.translateX(p, box.width(c)),
  [location.xyToString(location.BOTTOM_RIGHT)]: (p, c) =>
    xy.translateX(p, -box.width(c)),
  [location.xyToString(location.BOTTOM_LEFT)]: (p, c) => xy.translateX(p, box.width(c)),
};

const bestLocation = <C extends location.Location>(
  container: box.Box,
  window: box.Box,
  options: C[],
): C => {
  for (const location of options) {
    const distance = Math.abs(box.loc(window, location) - box.loc(container, location));
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
    const container = box.construct(e.target as HTMLElement);
    const window = box.construct(document.documentElement);
    const parse = location.location.safeParse(cornerOrLocation);

    const chooseRemainingLocation = (first: location.Location): location.Location => {
      let preferences: location.Location[];
      if (first === "center") {
        preferences = OUTER_LOCATION_PREFERENCES;
      } else if (location.isX(first))
        preferences = ["center", ...Y_LOCATION_PREFERENCES];
      else preferences = ["center", ...X_LOCATION_PREFERENCES];
      return location.construct(bestLocation(container, window, preferences));
    };

    let xy: location.XY = location.CENTER;
    if (parse.success) {
      xy = location.constructXY(parse.data, chooseRemainingLocation(parse.data));
    } else if (cornerOrLocation != null) {
      const v = cornerOrLocation as Partial<location.XY>;
      if (v.x == null && v.y != null)
        v.x = chooseRemainingLocation(location.construct(v.y)) as location.X;
      else if (v.y == null && v.x != null)
        v.y = chooseRemainingLocation(location.construct(v.x)) as location.Y;
      else if (v.x == null && v.y == null) {
        v.x = bestLocation(container, window, LOCATION_PREFERENCES) as location.X;
        v.y = chooseRemainingLocation(location.construct(v.x)) as location.Y;
      }
      xy = location.constructXY(v as location.XY);
    } else {
      const chosen = bestLocation(container, window, LOCATION_PREFERENCES);
      xy = location.constructXY(chosen, chooseRemainingLocation(chosen));
    }

    let pos = box.xyLoc(container, xy);
    const translate = LOCATION_TRANSLATIONS[location.xyToString(xy)];
    if (translate != null) pos = translate(pos, container);

    setState({
      location: xy,
      position: pos,
      elDims: box.dims(container),
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
              ...xy.css(state.position),
              // @ts-expect-error - css
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
