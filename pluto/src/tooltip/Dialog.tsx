// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tooltip/Dialog.css";

import {
  box,
  type CrudeTimeSpan,
  type destructor,
  type dimensions,
  location,
  TimeSpan,
  xy,
} from "@synnaxlabs/x";
import {
  cloneElement,
  type ComponentPropsWithoutRef,
  isValidElement,
  type ReactElement,
  type ReactNode,
  useCallback,
  useId,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { isRenderProp, type RenderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { useCombinedStateAndRef } from "@/hooks";
import { Text } from "@/text";
import { useConfig } from "@/tooltip/Config";

interface ChildProps {
  id?: string;
  onMouseDown?: (e: React.MouseEvent) => void;
  onMouseEnter?: (e: React.MouseEvent) => void;
  onMouseLeave?: (e: React.MouseEvent) => void;
}

export interface DialogProps extends Omit<ComponentPropsWithoutRef<"div">, "children"> {
  delay?: CrudeTimeSpan;
  location?: location.Outer | Partial<location.XY>;
  hide?: boolean;
  children: [ReactNode | RenderProp<ContentProps>, ReactElement<ChildProps>];
}

interface State {
  location: location.XY;
  position: xy.XY;
  triggerDims: dimensions.Dimensions;
}

export interface ContentProps extends State {}

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
  [location.xyToString(location.TOP_RIGHT)]: (p, c) =>
    xy.translate(p, "x", -box.width(c)),
  [location.xyToString(location.TOP_LEFT)]: (p, c) =>
    xy.translate(p, "x", box.width(c)),
  [location.xyToString(location.BOTTOM_RIGHT)]: (p, c) =>
    xy.translate(p, "x", -box.width(c)),
  [location.xyToString(location.BOTTOM_LEFT)]: (p, c) =>
    xy.translate(p, "x", box.width(c)),
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

export const chooseLocation = (
  cornerOrLocation: location.Outer | Partial<location.XY> | undefined,
  target: box.Box,
  window: box.Box,
): location.XY => {
  const parse = location.locationZ.safeParse(cornerOrLocation);
  const chooseRemainingLocation = (first: location.Location): location.Location => {
    let preferences: location.Location[];
    if (first === "center") preferences = OUTER_LOCATION_PREFERENCES;
    else if (location.isX(first)) preferences = ["center", ...Y_LOCATION_PREFERENCES];
    else preferences = ["center", ...X_LOCATION_PREFERENCES];
    return location.construct(bestLocation(target, window, preferences));
  };

  if (parse.success)
    return location.constructXY(parse.data, chooseRemainingLocation(parse.data));
  if (cornerOrLocation != null) {
    const v = { ...(cornerOrLocation as Partial<location.XY>) };
    if (v.x == null && v.y != null)
      v.x = chooseRemainingLocation(location.construct(v.y)) as location.X;
    else if (v.y == null && v.x != null)
      v.y = chooseRemainingLocation(location.construct(v.x)) as location.Y;
    else if (v.x == null && v.y == null) {
      v.x = bestLocation(target, window, LOCATION_PREFERENCES) as location.X;
      v.y = chooseRemainingLocation(location.construct(v.x)) as location.Y;
    }
    const l = location.constructXY(v as location.XY);
    return l;
  }
  const chosen = bestLocation(target, window, LOCATION_PREFERENCES);
  return location.constructXY(chosen, chooseRemainingLocation(chosen));
};

const resolveTarget = (target: HTMLElement, id: string): HTMLElement => {
  // we want to find the first parent that has the given id
  let el: HTMLElement | null = target;
  while (el != null) {
    if (el.id === id) return el;
    el = el.parentElement;
  }
  return target;
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
  const { startAccelerating, delay: configDelay } = useConfig();
  const parsedDelay = new TimeSpan(delay ?? configDelay);
  const [state, setState, stateRef] = useCombinedStateAndRef<State | null>(null);
  const [loadCLS, setLoadCLS] = useState<string>("");
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const id = useId();
  const visibleCleanup = useRef<destructor.Destructor | null>(null);
  const updateCLSTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const setStateAndLoadCLS = useCallback((s: State | null): void => {
    if ((s == null && stateRef.current == null) || updateCLSTimeoutRef.current != null)
      return;
    if (s != null) {
      setState(s);
      updateCLSTimeoutRef.current = setTimeout(() => {
        setLoadCLS(CSS.M("loaded"));
        updateCLSTimeoutRef.current = null;
      }, 1);
    } else {
      setLoadCLS("");
      updateCLSTimeoutRef.current = setTimeout(() => {
        setState(null);
        updateCLSTimeoutRef.current = null;
      }, 500);
    }
  }, []);

  const handleVisibleChange = useCallback(
    (e: React.MouseEvent, visible: boolean): void => {
      if (!visible || hide) {
        visibleCleanup.current?.();
        return setStateAndLoadCLS(null);
      }
      startAccelerating();
      const targetBox = box.construct(resolveTarget(e.target as HTMLElement, id));
      if (!box.contains(targetBox, xy.construct(e))) {
        visibleCleanup.current?.();
        setStateAndLoadCLS(null);
      }
      const window = box.construct(document.documentElement);
      const xyLoc = chooseLocation(cornerOrLocation, targetBox, window);

      let pos = box.xyLoc(targetBox, xyLoc);
      const translate = LOCATION_TRANSLATIONS[location.xyToString(xyLoc)];
      if (translate != null) pos = translate(pos, targetBox);

      const root = box.construct(document.body);
      setStateAndLoadCLS({
        location: xyLoc,
        position: xy.translate(pos, xy.scale(box.topLeft(root), -1)),
        triggerDims: box.dims(targetBox),
      });

      visibleCleanup.current?.();
      const handleMove = (e: MouseEvent): void => {
        const cursor = xy.construct(e);
        if (box.contains(targetBox, cursor)) return;
        setStateAndLoadCLS(null);
        document.removeEventListener("mousemove", handleMove);
        visibleCleanup.current = null;
        if (timeoutRef.current != null) clearTimeout(timeoutRef.current);
      };

      document.addEventListener("mousemove", handleMove);
      visibleCleanup.current = () =>
        document.removeEventListener("mousemove", handleMove);
      document.addEventListener(
        "mousedown",
        () => {
          setStateAndLoadCLS(null);
          visibleCleanup.current?.();
        },
        { once: true },
      );
    },
    [startAccelerating, cornerOrLocation, hide, id, parsedDelay.milliseconds],
  );

  // TODO: fix this in-component state update.
  if (hide && state != null) setStateAndLoadCLS(null);

  const handleMouseEnter = useCallback(
    (e: React.MouseEvent): void => {
      timeoutRef.current = setTimeout(
        () => handleVisibleChange(e, true),
        parsedDelay.milliseconds,
      );
    },
    [handleVisibleChange, parsedDelay.milliseconds],
  );

  const handleMouseLeave = useCallback(
    (e: React.MouseEvent): void => {
      if (timeoutRef.current != null) clearTimeout(timeoutRef.current);
      handleVisibleChange(e, false);
    },
    [handleVisibleChange],
  );

  const [tip, children_] = children;

  const root = document.body;

  return (
    <>
      {state != null &&
        createPortal(
          <div
            key={id}
            className={CSS(
              CSS.B("tooltip"),
              CSS.loc(state.location.x),
              CSS.loc(state.location.y),
              loadCLS,
            )}
            style={{
              [CSS.var("pos-x")]: CSS.px(state.position.x),
              [CSS.var("pos-y")]: CSS.px(state.position.y),
            }}
          >
            {isRenderProp(tip) ? tip(state) : formatTip(tip)}
          </div>,
          root,
        )}
      {cloneElement(children_, {
        onMouseEnter: (e) => {
          handleMouseEnter(e);
          children_.props.onMouseEnter?.(e);
        },
        onMouseLeave: (e) => {
          handleMouseLeave(e);
          children_.props.onMouseLeave?.(e);
        },
        onMouseDown: useCallback(
          (e: React.MouseEvent) => {
            handleMouseLeave(e);
            children_.props.onMouseDown?.(e);
          },
          [handleVisibleChange],
        ),
      })}
    </>
  );
};

export const formatTip = (tip: ReactNode): ReactNode => {
  if (typeof tip === "string" || typeof tip === "number" || !isValidElement(tip))
    return (
      <Text.Text level="small" color={11}>
        {tip as string | number}
      </Text.Text>
    );
  return tip;
};
