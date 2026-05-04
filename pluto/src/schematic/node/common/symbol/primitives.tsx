// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/symbol/primitives.css";

import { color, dimensions, direction, type location, TimeSpan } from "@synnaxlabs/x";
import {
  Handle as RFHandle,
  type HandleProps as RFHandleProps,
  Position as RFPosition,
  useUpdateNodeInternals,
} from "@xyflow/react";
import {
  type ComponentPropsWithoutRef,
  type ComponentPropsWithRef,
  type MouseEventHandler,
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useMemo,
  useRef,
} from "react";

import { CSS } from "@/css";
import { Theming } from "@/theming";
import { stopPropagation } from "@/util/event";

export interface PathProps extends ComponentPropsWithoutRef<"path"> {}

export const Path = (props: PathProps): ReactElement => (
  <path vectorEffect="non-scaling-stroke" {...props} />
);

export interface RectProps extends ComponentPropsWithoutRef<"rect"> {}

export const Rect = (props: RectProps): ReactElement => (
  <rect vectorEffect="non-scaling-stroke" {...props} />
);

export interface CircleProps extends ComponentPropsWithoutRef<"circle"> {}

export const Circle = (props: CircleProps): ReactElement => (
  <circle vectorEffect="non-scaling-stroke" {...props} />
);

export interface LineProps extends ComponentPropsWithoutRef<"line"> {}

export const Line = (props: LineProps): ReactElement => (
  <line vectorEffect="non-scaling-stroke" {...props} />
);

export const ORIENTATION_RF_POSITIONS: Record<
  location.Outer,
  Record<location.Outer, RFPosition>
> = {
  left: {
    left: RFPosition.Left,
    right: RFPosition.Right,
    top: RFPosition.Top,
    bottom: RFPosition.Bottom,
  },
  right: {
    left: RFPosition.Right,
    right: RFPosition.Left,
    top: RFPosition.Bottom,
    bottom: RFPosition.Top,
  },
  top: {
    left: RFPosition.Bottom,
    right: RFPosition.Top,
    top: RFPosition.Left,
    bottom: RFPosition.Right,
  },
  bottom: {
    left: RFPosition.Top,
    right: RFPosition.Bottom,
    top: RFPosition.Right,
    bottom: RFPosition.Left,
  },
};

export const smartPosition = (
  position: location.Outer,
  orientation: location.Outer,
): RFPosition => ORIENTATION_RF_POSITIONS[orientation][position];

export const swapRF = (position: RFPosition, bypass: boolean = false): RFPosition => {
  if (bypass) return position;
  switch (position) {
    case RFPosition.Left:
      return RFPosition.Right;
    case RFPosition.Right:
      return RFPosition.Left;
    case RFPosition.Top:
      return RFPosition.Bottom;
    case RFPosition.Bottom:
      return RFPosition.Top;
    default:
      return RFPosition.Top;
  }
};

export const adjustHandle = (
  top: number,
  left: number,
  orientation: location.Outer,
  prevent: boolean = false,
): { left: number; top: number } => {
  if (prevent) return { left, top };
  if (orientation === "left") return { top, left };
  if (orientation === "right") return { top: 100 - top, left: 100 - left };
  if (orientation === "top") return { top: 100 - left, left: top };
  return { top: left, left: 100 - top };
};

export interface OrientableProps {
  orientation?: location.Outer;
}

export interface SmartHandlesProps extends PropsWithChildren<{}> {
  orientation: location.Outer;
  refreshDeps?: unknown;
}

export const HandleBoundary = ({
  children,
  orientation,
  refreshDeps,
}: SmartHandlesProps): ReactElement | null => {
  let updateInternals: ReturnType<typeof useUpdateNodeInternals> | undefined;
  try {
    updateInternals = useUpdateNodeInternals();
  } catch {
    return null;
  }
  const ref = useRef<HTMLDivElement & HTMLButtonElement>(null);
  const first = useRef<boolean>(true);
  useEffect(() => {
    if (ref.current == null) return;
    if (first.current) {
      first.current = false;
      return;
    }
    const node = ref.current.closest(".react-flow__node");
    const id = node?.getAttribute("data-id");
    if (id == null) return;
    updateInternals?.(id);
  }, [orientation, refreshDeps]);
  return (
    <>
      <span ref={ref} />
      {children}
    </>
  );
};

export interface HandleProps extends Omit<RFHandleProps, "type" | "position"> {
  orientation: location.Outer;
  location: location.Outer;
  position?: RFPosition;
  preventAutoAdjust?: boolean;
  swap?: boolean;
  left: number;
  top: number;
  id: string;
}

export const Handle = ({
  location,
  orientation,
  preventAutoAdjust,
  left,
  swap,
  top,
  style,
  ...rest
}: HandleProps): ReactElement => {
  const adjusted = adjustHandle(top, left, orientation, preventAutoAdjust);
  return (
    <RFHandle
      position={swapRF(smartPosition(location, orientation), !swap)}
      {...rest}
      type="source"
      onClick={stopPropagation}
      className={(CSS.B("handle"), CSS.BE("handle", rest.id))}
      style={{
        left: `${adjusted.left}%`,
        top: `${adjusted.top}%`,
        ...style,
      }}
    />
  );
};

export interface ToggleProps extends Omit<
  ComponentPropsWithRef<"button">,
  "color" | "value"
> {
  triggered?: boolean;
  enabled?: boolean;
  color?: color.Crude;
  onClickDelay?: number | TimeSpan;
}

export interface ToggleValveButtonProps extends ToggleProps, OrientableProps {}

export const Toggle = ({
  className,
  enabled = false,
  triggered = false,
  orientation = "left",
  color: colorVal,
  onClickDelay = 0,
  onClick,
  onMouseDown,
  style,
  ...rest
}: ToggleValveButtonProps): ReactElement => {
  const parsedDelay = TimeSpan.fromMilliseconds(onClickDelay);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleClick: MouseEventHandler<HTMLButtonElement> = (e) => {
    if (parsedDelay.isZero) onClick?.(e);
  };

  const handleMouseDown: MouseEventHandler<HTMLButtonElement> = (e) => {
    onMouseDown?.(e);
    if (parsedDelay.isZero) return;
    document.addEventListener(
      "mouseup",
      () => {
        if (timeoutRef.current != null) clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      },
      { once: true },
    );
    timeoutRef.current = setTimeout(() => {
      onClick?.(e);
      timeoutRef.current = null;
    }, parsedDelay.milliseconds);
  };

  const pStyle = useMemo(() => {
    if (parsedDelay.isZero) return style;
    return {
      ...style,
      [CSS.var("toggle-delay")]: `${parsedDelay.seconds.toString()}s`,
    };
  }, [parsedDelay.milliseconds, style]);

  return (
    <button
      className={CSS(
        CSS.B("symbol-primitive"),
        CSS.B("symbol-primitive-toggle"),
        !parsedDelay.isZero && CSS.BM("symbol-primitive-toggle", "delayed"),
        orientation != null && CSS.loc(orientation),
        enabled && CSS.M("enabled"),
        triggered && CSS.M("triggered"),
        className,
      )}
      color={color.cssString(colorVal)}
      onClick={handleClick}
      onMouseDown={handleMouseDown}
      style={pStyle}
      {...rest}
    />
  );
};

export interface DivProps
  extends Omit<ComponentPropsWithRef<"div">, "color" | "onResize">, OrientableProps {}

export const Div = ({ className, ...rest }: DivProps): ReactElement => (
  <div className={CSS(CSS.B("symbol-primitive"), className)} {...rest} />
);

export interface SVGBasedPrimitiveProps extends OrientableProps {
  color?: color.Crude;
  scale?: number;
}

export interface InternalSVGProps
  extends
    SVGBasedPrimitiveProps,
    Omit<
      ComponentPropsWithoutRef<"svg">,
      "direction" | "color" | "orientation" | "scale"
    > {
  dimensions: dimensions.Dimensions;
}

export const BASE_SCALE = 0.8;

export const InternalSVG = ({
  dimensions: dims,
  orientation = "left",
  children,
  className,
  color: colorVal,
  style = {},
  scale = 1,
  ...rest
}: InternalSVGProps): ReactElement => {
  const dir = direction.construct(orientation);
  dims = dir === "y" ? dimensions.swap(dims) : dims;
  const colorStr = color.cssString(colorVal);
  const theme = Theming.use();
  let pStyle = {
    ...style,
    aspectRatio: `${dims.width} / ${dims.height}`,
    width: dimensions.scale(dims, scale * BASE_SCALE).width,
  };
  if (colorVal != null)
    pStyle = {
      ...pStyle,
      [CSS.var("symbol-color")]: color.rgbString(colorVal),
      [CSS.var("symbol-color-contrast")]: color.rgbString(
        color.pickByContrast(colorVal, theme.colors.gray.l0, theme.colors.gray.l11),
      ),
    };

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox={dimensions.svgViewBox(dims)}
      className={CSS(CSS.loc(orientation), className)}
      fill={colorStr}
      stroke={colorStr}
      {...rest}
      style={pStyle}
    >
      <g>{children}</g>
    </svg>
  );
};
