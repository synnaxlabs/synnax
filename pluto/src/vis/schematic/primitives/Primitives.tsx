// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/schematic/primitives/Primitives.css";

import { dimensions, direction, type location, xy } from "@synnaxlabs/x";
import {
  type ComponentPropsWithoutRef,
  CSSProperties,
  type MouseEventHandler,
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";
import {
  Handle as RFHandle,
  type HandleProps as RFHandleProps,
  Position as RFPosition,
  useUpdateNodeInternals,
} from "reactflow";

import { Align } from "@/align";
import { Button as CoreButton } from "@/button";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Input } from "@/input";
import { Text } from "@/text";
import { Theming } from "@/theming";

interface PathProps extends ComponentPropsWithoutRef<"path"> {}

const Path = (props: PathProps): ReactElement => (
  <path vectorEffect="non-scaling-stroke" {...props} />
);

interface RectProps extends ComponentPropsWithoutRef<"rect"> {}

const Rect = (props: RectProps): ReactElement => (
  <rect vectorEffect="non-scaling-stroke" {...props} />
);

interface CircleProps extends ComponentPropsWithoutRef<"circle"> {}

const Circle = (props: CircleProps): ReactElement => (
  <circle vectorEffect="non-scaling-stroke" {...props} />
);

interface LineProps extends ComponentPropsWithoutRef<"line"> {}

const Line = (props: LineProps): ReactElement => (
  <line vectorEffect="non-scaling-stroke" {...props} />
);

const ORIENTATION_RF_POSITIONS: Record<
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

const smartPosition = (
  position: location.Outer,
  orientation: location.Outer,
): RFPosition => ORIENTATION_RF_POSITIONS[orientation][position];

const swapRF = (position: RFPosition, bypass: boolean = false): RFPosition => {
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

const adjustHandle = (
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

interface OrientableProps {
  orientation?: location.Outer;
}

interface SmartHandlesProps extends PropsWithChildren<{}>, OrientableProps {}

const HandleBoundary = ({ children, orientation }: SmartHandlesProps): ReactElement => {
  let updateInternals: ReturnType<typeof useUpdateNodeInternals> | undefined;
  try {
    updateInternals = useUpdateNodeInternals();
  } catch {
    return <></>;
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
  }, [orientation]);
  return (
    <>
      <span ref={ref} />
      {children}
    </>
  );
};

interface HandleProps extends Omit<RFHandleProps, "type" | "position"> {
  orientation: location.Outer;
  location: location.Outer;
  position?: RFPosition;
  preventAutoAdjust?: boolean;
  swap?: boolean;
  left: number;
  top: number;
  id: string;
}

const Handle = ({
  location,
  orientation,
  preventAutoAdjust,
  left,
  swap,
  top,
  ...props
}: HandleProps): ReactElement => {
  const adjusted = adjustHandle(top, left, orientation, preventAutoAdjust);
  return (
    <RFHandle
      position={swapRF(smartPosition(location, orientation), !swap)}
      {...props}
      type="source"
      onClick={(e) => e.stopPropagation()}
      className={(CSS.B("handle"), CSS.BE("handle", props.id))}
      style={{
        left: `${adjusted.left}%`,
        top: `${adjusted.top}%`,
      }}
    />
  );
};

interface ToggleProps
  extends Omit<ComponentPropsWithoutRef<"button">, "color" | "value"> {
  triggered?: boolean;
  enabled?: boolean;
  color?: Color.Crude;
}

interface ToggleValveButtonProps extends ToggleProps, OrientableProps {}

const Toggle = ({
  className,
  enabled = false,
  triggered = false,
  orientation = "left",
  color,
  ...props
}: ToggleValveButtonProps): ReactElement => (
  <button
    className={CSS(
      CSS.B("symbol-primitive"),
      CSS.B("symbol-primitive-toggle"),
      orientation != null && CSS.loc(orientation),
      enabled && CSS.M("enabled"),
      triggered && CSS.M("triggered"),
      className,
    )}
    color={Color.cssString(color)}
    {...props}
  />
);

interface DivProps
  extends Omit<ComponentPropsWithoutRef<"div">, "color" | "onResize">,
    OrientableProps {}

const Div = ({ className, ...props }: DivProps): ReactElement => (
  <div className={CSS(CSS.B("symbol-primitive"), className)} {...props} />
);

interface SVGBasedPrimitiveProps extends OrientableProps {
  color?: Color.Crude;
  scale?: number;
}

interface InternalSVGProps
  extends SVGBasedPrimitiveProps,
    Omit<
      ComponentPropsWithoutRef<"svg">,
      "direction" | "color" | "orientation" | "scale"
    > {
  dimensions: dimensions.Dimensions;
}

const BASE_SCALE = 0.8;

const InternalSVG = ({
  dimensions: dims,
  orientation = "left",
  children,
  className,
  color,
  style = {},
  scale = 1,
  ...props
}: InternalSVGProps): ReactElement => {
  const dir = direction.construct(orientation);
  dims = dir === "y" ? dimensions.swap(dims) : dims;
  const colorStr = Color.cssString(color);
  const theme = Theming.use();
  if (color != null) {
    // @ts-expect-error - css variables
    style[CSS.var("symbol-color")] = new Color.Color(color).rgbString;
    // @ts-expect-error - css variables
    style[CSS.var("symbol-color-contrast")] = new Color.Color(color).pickByContrast(
      theme.colors.gray.l0,
      theme.colors.gray.l10,
    ).rgbString;
  }
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox={dimensions.svgViewBox(dims)}
      className={CSS(CSS.loc(orientation), className)}
      fill={colorStr}
      stroke={colorStr}
      {...props}
      style={{
        aspectRatio: `${dims.width} / ${dims.height}`,
        width: dimensions.scale(dims, scale * BASE_SCALE).width,
        ...style,
      }}
    >
      <g>{children}</g>
    </svg>
  );
};

export interface FourWayValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const FourWayValve = ({
  className,
  orientation,
  scale,
  color,
  ...props
}: FourWayValveProps): ReactElement => (
  <Toggle
    {...props}
    orientation={orientation}
    className={CSS(CSS.B("four-way-valve"), className)}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation="left" left={2.2989} top={50} id="1" />
      <Handle location="right" orientation="left" left={97.7011} top={50} id="2" />
      <Handle location="top" orientation="left" left={50} top={2.2989} id="3" />
      <Handle location="bottom" orientation="left" left={50} top={97.7011} id="4" />
    </HandleBoundary>
    <InternalSVG dimensions={{ width: 87, height: 87 }} color={color} scale={scale}>
      <Path d="M43.5 43.5L6.35453 24.7035C4.35901 23.6937 2 25.1438 2 27.3803V59.6197C2 61.8562 4.35901 63.3063 6.35453 62.2965L43.5 43.5ZM43.5 43.5L80.6455 24.7035C82.641 23.6937 85 25.1438 85 27.3803V59.6197C85 61.8562 82.641 63.3063 80.6455 62.2965L43.5 43.5Z" />
      <Path d="M43.5 43.5L24.7035 80.6455C23.6937 82.641 25.1438 85 27.3803 85H59.6197C61.8562 85 63.3063 82.641 62.2965 80.6455L43.5 43.5ZM43.5 43.5L24.7035 6.35453C23.6937 4.35901 25.1438 2 27.3803 2H59.6197C61.8562 2 63.3063 4.35901 62.2965 6.35453L43.5 43.5Z" />
    </InternalSVG>
  </Toggle>
);

export interface ThreeWayValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const ThreeWayValve = ({
  color,
  orientation = "left",
  scale,
  ...props
}: ThreeWayValveProps): ReactElement => (
  <Toggle
    {...props}
    className={CSS(CSS.B("three-way-valve"))}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={97.0278}
        id="1"
      />
      <Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={33.1308}
        id="2"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={33.1308}
        id="3"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M43.5 21L6.35453 2.20349C4.35901 1.19372 2 2.64384 2 4.88029V37.1197C2 39.3562 4.35901 40.8063 6.35453 39.7965L43.5 21ZM43.5 21L80.6455 2.20349C82.641 1.19372 85 2.64384 85 4.8803V37.1197C85 39.3562 82.641 40.8063 80.6455 39.7965L43.5 21Z" />
      <Path d="M44.3923 22.3611C44.0222 21.6298 42.9778 21.6298 42.6077 22.3611L24.7035 57.7433C23.6937 59.7388 25.1438 62.0978 27.3803 62.0978L59.6197 62.0978C61.8562 62.0978 63.3063 59.7388 62.2965 57.7433L44.3923 22.3611Z" />
    </InternalSVG>
  </Toggle>
);

export interface ValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const Valve = ({
  orientation = "left",
  color,
  scale,
  ...props
}: ValveProps): ReactElement => (
  <Toggle {...props}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={2.2989} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7701}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 42 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M43.5 21L6.35453 2.20349C4.35901 1.19372 2 2.64384 2 4.88029V37.1197C2 39.3562 4.35901 40.8063 6.35453 39.7965L43.5 21ZM43.5 21L80.6455 2.20349C82.641 1.19372 85 2.64384 85 4.8803V37.1197C85 39.3562 82.641 40.8063 80.6455 39.7965L43.5 21Z" />
    </InternalSVG>
  </Toggle>
);

export interface SolenoidValveProps extends ToggleProps, SVGBasedPrimitiveProps {
  normallyOpen?: boolean;
}

export const SolenoidValve = ({
  className,
  color,
  orientation = "left",
  normallyOpen = false,
  scale,
  ...props
}: SolenoidValveProps): ReactElement => (
  <Toggle
    className={CSS(
      CSS.B("solenoid-valve"),
      normallyOpen && CSS.M("normally-open"),
      className,
    )}
    {...props}
  >
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={69.5652}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={69.5652}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 69 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path
        className={CSS.B("body")}
        d="M43.5 48L6.35453 29.2035C4.35901 28.1937 2 29.6438 2 31.8803V64.1197C2 66.3562 4.35901 67.8063 6.35453 66.7965L43.5 48ZM43.5 48L80.6455 29.2035C82.641 28.1937 85 29.6438 85 31.8803V64.1197C85 66.3562 82.641 67.8063 80.6455 66.7965L43.5 48Z"
      />
      <Line x1={43.5} x2={43.5} y1={24.5333} y2={48} />
      <Rect x="29" y="2" width="29" height="22.5333" rx="1" />
    </InternalSVG>
  </Toggle>
);

export interface ReliefValveProps extends DivProps, SVGBasedPrimitiveProps {}

export const ReliefValve = ({
  className,
  orientation = "left",
  color,
  scale,
  ...props
}: ReliefValveProps): ReactElement => (
  <Div
    orientation={orientation}
    className={CSS(CSS.B("relief-valve"), className)}
    {...props}
  >
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={63.7931}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={63.7931}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 58 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M43.5 37L6.35453 18.2035C4.35901 17.1937 2 18.6438 2 20.8803V53.1197C2 55.3562 4.35901 56.8063 6.35453 55.7965L43.5 37ZM43.5 37L80.6455 18.2035C82.641 17.1937 85 18.6438 85 20.8803V53.1197C85 55.3562 82.641 56.8063 80.6455 55.7965L43.5 37Z" />
      <Path d="M43.5 2 L43.5 37" strokeLinecap="round" />
      <Path d="M31.8011 14.0802L55.1773 4.29611" strokeLinecap="round" />
      <Path d="M31.8011 20.0802L55.1773 10.2961" strokeLinecap="round" />
      <Path d="M31.8011 26.0802L55.1773 16.2961" strokeLinecap="round" />
    </InternalSVG>
  </Div>
);

export interface CheckValveProps extends DivProps, SVGBasedPrimitiveProps {}

export const CheckValve = ({
  className,
  orientation = "left",
  color,
  scale,
  ...props
}: CheckValveProps): ReactElement => (
  <Div
    orientation={orientation}
    className={CSS(CSS.B("check-valve"), className)}
    {...props}
  >
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={4.2222}
        top={48.8372}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={94.2038}
        top={48.8372}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 45, height: 43 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1="42.3917" y1="2" x2="42.3917" y2="41" strokeLinecap="round" />
      <Path d="M41.6607 21.8946C42.3917 21.5238 42.3906 20.4794 41.6589 20.1101L6.25889 2.2412C4.26237 1.23341 1.90481 2.68589 1.90704 4.92235L1.93925 37.1617C1.94148 39.3982 4.30194 40.846 6.29644 39.8342L41.6607 21.8946Z" />
    </InternalSVG>
  </Div>
);

export interface AngledValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const AngledValve = ({
  color,
  className,
  orientation = "left",
  scale,
  ...props
}: AngledValveProps): ReactElement => (
  <Toggle
    {...props}
    orientation={orientation}
    className={CSS(CSS.B("angled-valve"), className)}
  >
    <HandleBoundary orientation={orientation}>
      <Handle
        location="bottom"
        orientation={orientation}
        left={32.8125}
        top={97.0278}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.0278}
        top={32.8125}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M22.3611 20.1077C21.6298 20.4778 21.6298 21.5222 22.3611 21.8923L57.7433 39.7965C59.7388 40.8063 62.0978 39.3562 62.0978 37.1197L62.0978 4.88029C62.0978 2.64384 59.7388 1.19372 57.7433 2.2035L22.3611 20.1077Z" />
      <Path d="M21.8923 22.3611C21.5222 21.6298 20.4778 21.6298 20.1077 22.3611L2.20349 57.7433C1.19372 59.7388 2.64384 62.0978 4.8803 62.0978L37.1197 62.0978C39.3562 62.0978 40.8063 59.7388 39.7965 57.7433L21.8923 22.3611Z" />
    </InternalSVG>
  </Toggle>
);

export interface PumpProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const Pump = ({
  color,
  className,
  orientation = "left",
  scale,
  ...props
}: PumpProps): ReactElement => (
  <Toggle
    {...props}
    className={CSS(CSS.B("pump"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.125} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.875}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="32" cy="32" r="30" />
      <Line
        x1="32"
        y1="2"
        x2="62"
        y2="32"
        className={CSS(CSS.M("detail"), className)}
      />
      <Line
        x1="32"
        y1="62"
        x2="62"
        y2="32"
        className={CSS(CSS.M("detail"), className)}
      />
    </InternalSVG>
  </Toggle>
);

export interface BurstDiscProps extends DivProps, SVGBasedPrimitiveProps {}

export const BurstDisc = ({
  className,
  color,
  orientation = "left",
  scale,
  ...props
}: BurstDiscProps): ReactElement => {
  const colorStr = Color.cssString(color);
  return (
    <Div {...props} className={CSS(CSS.B("symbol"), className)}>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={5} top={50} id="1" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 40, height: 48 }}
        color={color}
        orientation={orientation}
        scale={scale}
      >
        <Path d="M24 24C24 35.1852 15.2546 44.4725 3.87626 45.8297C2.90571 45.9455 2 45.1407 2 44V4C2 2.85926 2.90571 2.0545 3.87626 2.17027C15.2546 3.52755 24 12.8148 24 24Z" />
        <Path
          d="M37.9706 23.2076C38.4906 23.6079 38.4906 24.3921 37.9706 24.7924L33.86 27.9568C33.2024 28.463 32.25 27.9942 32.25 27.1644V20.8356C32.25 20.0058 33.2024 19.537 33.86 20.0432L37.9706 23.2076Z"
          fill={colorStr}
        />
        <Path d="M33 24H2" strokeLinecap="round" />
      </InternalSVG>
    </Div>
  );
};

export interface CapProps extends SVGBasedPrimitiveProps, DivProps {}

export const Cap = ({
  className,
  orientation = "left",
  color,
  scale,
  ...props
}: CapProps): ReactElement => (
  <Div className={CSS(CSS.B("cap"), className)} {...props}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={7.6923} top={50} id="1" />
    </HandleBoundary>
    <InternalSVG
      color={color}
      dimensions={{ width: 26, height: 48 }}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M24 24C24 35.1852 15.2546 44.4725 3.87626 45.8297C2.90571 45.9455 2 45.1407 2 44V4C2 2.85926 2.90571 2.0545 3.87626 2.17027C15.2546 3.52755 24 12.8148 24 24Z" />
    </InternalSVG>
  </Div>
);

export interface ManualValveProps extends SVGBasedPrimitiveProps, DivProps {}

export const ManualValve = ({
  className,
  orientation = "left",
  color,
  scale,
  ...props
}: ManualValveProps): ReactElement => (
  <Div className={CSS(CSS.B("manual-valve"), className)} {...props}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={56.25}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={56.25}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 48 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1="43.5" y1="27" x2="43.5" y2="1" />
      <Path d="M19.64 1 L66.68 1" strokeLinecap="round" />
      <Path d="M43.5 27L6.35453 8.20349C4.35901 7.19372 2 8.64384 2 10.8803V43.1197C2 45.3562 4.35901 46.8063 6.35453 45.7965L43.5 27ZM43.5 27L80.6455 8.20349C82.641 7.19372 85 8.64384 85 10.8803V43.1197C85 45.3562 82.641 46.8063 80.6455 45.7965L43.5 27Z" />
    </InternalSVG>
  </Div>
);

export interface FilterProps extends SVGBasedPrimitiveProps, DivProps {}

export const Filter = ({
  className,
  orientation = "left",
  color,
  scale,
  ...props
}: FilterProps): ReactElement => (
  <Div className={CSS(CSS.B("filter"), className)} {...props}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={11.5385}
        top={50}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={88.4615}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 52, height: 34 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M6 17L24.8 2.9C25.5111 2.36667 26.4889 2.36667 27.2 2.9L46 17M6 17L24.8 31.1C25.5111 31.6333 26.4889 31.6333 27.2 31.1L46 17" />
    </InternalSVG>
  </Div>
);

type DetailedBorderRadius = Record<location.CornerXYString, xy.XY>;

type BorderRadius =
  | number
  | Record<direction.Direction, number>
  | Record<location.CornerXYString, number>
  | DetailedBorderRadius;

const parseBorderRadius = (radius: BorderRadius): DetailedBorderRadius => {
  if (typeof radius === "number")
    return {
      topLeft: xy.construct(radius),
      topRight: xy.construct(radius),
      bottomLeft: xy.construct(radius),
      bottomRight: xy.construct(radius),
    };
  if ("x" in radius)
    return {
      topLeft: radius,
      topRight: radius,
      bottomLeft: radius,
      bottomRight: radius,
    };
  if (typeof radius.bottomLeft === "number")
    return {
      topLeft: xy.construct(radius.topLeft),
      topRight: xy.construct(radius.topRight),
      bottomLeft: xy.construct(radius.bottomLeft),
      bottomRight: xy.construct(radius.bottomRight),
    };
  return radius as DetailedBorderRadius;
};

const cssBorderRadius = (radius: DetailedBorderRadius): string => {
  const { topLeft, topRight, bottomLeft, bottomRight } = radius;
  return `${topLeft.x}% ${topRight.x}% ${bottomRight.x}% ${bottomLeft.x}% / ${topLeft.y}% ${topRight.y}% ${bottomRight.y}% ${bottomLeft.y}%`;
};

const DEFAULT_DIMENSIONS = { width: 40, height: 80 };
export const DEFAULT_BORDER_RADIUS = { x: 50, y: 10 };

export interface TankProps extends DivProps {
  dimensions?: dimensions.Dimensions;
  borderRadius?: BorderRadius;
  color?: Color.Crude;
  onResize?: (dimensions: dimensions.Dimensions) => void;
  boxBorderRadius?: number;
  backgroundColor?: Color.Crude;
}

export const Tank = ({
  className,
  dimensions = DEFAULT_DIMENSIONS,
  borderRadius = DEFAULT_BORDER_RADIUS,
  boxBorderRadius,
  color,
  backgroundColor,
  ...props
}: TankProps): ReactElement => {
  const detailedRadius = parseBorderRadius(borderRadius);
  const hasCornerBoundaries = boxBorderRadius == null;
  const t = Theming.use();
  return (
    <Div
      className={CSS(className, CSS.B("tank"))}
      style={{
        ...dimensions,
        borderRadius: boxBorderRadius ?? cssBorderRadius(detailedRadius),
        borderColor: Color.cssString(color ?? t.colors.gray.l9),
        backgroundColor:
          backgroundColor == null ? undefined : Color.cssString(backgroundColor),
      }}
      {...props}
    >
      <HandleBoundary>
        <Handle location="top" orientation="left" left={50} top={0} id="1" />
        {hasCornerBoundaries && (
          <>
            <Handle
              location="top"
              orientation="left"
              left={-0.5}
              top={detailedRadius.topLeft.y - 1}
              id="2"
            />
            <Handle
              location="top"
              orientation="left"
              left={101}
              top={detailedRadius.topRight.y - 1}
              id="3"
            />
          </>
        )}
        <Handle location="bottom" orientation="left" left={50} top={100} id="4" />
        {hasCornerBoundaries && (
          <>
            <Handle
              location="bottom"
              orientation="left"
              left={-0.5}
              top={100 - detailedRadius.bottomLeft.y + 1}
              id="5"
            />
            <Handle
              location="bottom"
              orientation="left"
              left={101}
              top={100 - detailedRadius.bottomRight.y + 1}
              id="6"
            />
          </>
        )}
        <Handle location="left" orientation="left" left={-0.5} top={50} id="7" />
        <Handle location="right" orientation="left" left={101} top={50} id="8" />
      </HandleBoundary>
    </Div>
  );
};

export interface RegulatorProps extends DivProps, SVGBasedPrimitiveProps {}

export const Regulator = ({
  className,
  orientation = "left",
  color,
  scale,
  ...props
}: RegulatorProps): ReactElement => (
  <Div className={CSS(className, CSS.B("regulator"))} {...props}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={3.4091}
        top={66.25}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7273}
        top={66.25}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 88, height: 80 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M44.5 53L7.35453 34.2035C5.35901 33.1937 3 34.6438 3 36.8803V69.1197C3 71.3562 5.35901 72.8063 7.35453 71.7965L44.5 53ZM44.5 53L81.6455 34.2035C83.641 33.1937 86 34.6438 86 36.8803V69.1197C86 71.3562 83.641 72.8063 81.6455 71.7965L44.5 53Z" />
      <Path d="M61 30C62.6569 30 64.0231 28.6494 63.7755 27.0111C63.141 22.8129 61.181 18.8968 58.1421 15.8579C54.3914 12.1071 49.3043 10 44 10C38.6957 10 33.6086 12.1071 29.8579 15.8579C26.819 18.8968 24.859 22.8129 24.2245 27.0111C23.9769 28.6494 25.3431 30 27 30L44.5 30H61Z" />
      <Line x1="44.5" y1="53" x2="44.5" y2="30" strokeLinecap="round" />
      <Path d="M44.5 10V8C44.5 6.34315 45.3431 5 47 5H80C81.6569 5 83 6.34315 83 8V24.4281C83 25.4126 82.517 26.3344 81.7076 26.8947L44.5 53" />
    </InternalSVG>
  </Div>
);

export interface OrificeProps extends DivProps, SVGBasedPrimitiveProps {}

export const Orifice = ({
  className,
  orientation = "left",
  scale,
  color,
  ...props
}: OrificeProps): ReactElement => (
  <Div className={CSS(CSS.B("orifice"), className)} {...props}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={2.8571} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={97.1429}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 70, height: 34 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M2 2.5C27.451 21.3469 60.0915 12.0132 68 2.5" strokeLinecap="round" />
      <Path d="M2 32.5C27.451 13.6531 60.0915 22.9868 68 32.5" strokeLinecap="round" />
    </InternalSVG>
  </Div>
);

export interface NeedleValveProps extends DivProps, SVGBasedPrimitiveProps {}

export const NeedleValve = ({
  className,
  orientation = "left",
  color,
  scale,
  ...props
}: NeedleValveProps): ReactElement => (
  <Div className={CSS(CSS.B("needle-valve"), className)} {...props}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={51.1905}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={51.1905}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 42 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path
        d="M43.0152 21.5391L38.237 2.62245C38.1573 2.30658 38.396 2 38.7218 2L48.2782 2C48.604 2 48.8427 2.30658 48.763 2.62245L43.9848 21.5391C43.8576 22.0425 43.1424 22.0425 43.0152 21.5391Z"
        fill={Color.cssString(color)}
      />
      <Path d="M43.5 21.5L6.35453 2.70349C4.35901 1.69372 2 3.14384 2 5.38029V37.6197C2 39.8562 4.35901 41.3063 6.35453 40.2965L43.5 21.5ZM43.5 21.5L80.6455 2.70349C82.641 1.69372 85 3.14384 85 5.3803V37.6197C85 39.8562 82.641 41.3063 80.6455 40.2965L43.5 21.5Z" />
    </InternalSVG>
  </Div>
);

export interface AngledReliefValveProps extends DivProps, SVGBasedPrimitiveProps {}

export const AngledReliefValve = ({
  className,
  orientation = "left",
  color,
  scale,
  ...props
}: AngledReliefValveProps): ReactElement => (
  <Div
    orientation={orientation}
    className={CSS(CSS.B("angled-relief-valve"), className)}
    {...props}
  >
    <HandleBoundary>
      <Handle
        location="bottom"
        orientation={orientation}
        left={32.8125}
        top={97.5922}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.0278}
        top={45.5639}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 79 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Line x1={21} y1={2} x2={21} y2={36.7} strokeLinecap="round" />
      <Path d="M9.05106 14.0802L32.4273 4.29611" strokeLinecap="round" />
      <Path d="M9.05106 20.0802L32.4273 10.2961" strokeLinecap="round" />
      <Path d="M9.05106 26.0802L32.4273 16.2961" strokeLinecap="round" />
      <Path d="M22.3611 35.1077C21.6298 35.4778 21.6298 36.5222 22.3611 36.8923L57.7433 54.7965C59.7388 55.8063 62.0978 54.3562 62.0978 52.1197L62.0978 19.8803C62.0978 17.6438 59.7388 16.1937 57.7433 17.2035L22.3611 35.1077Z" />
      <Path d="M21.8923 37.3611C21.5222 36.6298 20.4778 36.6298 20.1077 37.3611L2.20349 72.7433C1.19372 74.7388 2.64384 77.0978 4.8803 77.0978H37.1197C39.3562 77.0978 40.8063 74.7388 39.7965 72.7433L21.8923 37.3611Z" />
    </InternalSVG>
  </Div>
);

export interface ValueProps extends DivProps {
  dimensions?: dimensions.Dimensions;
  color?: Color.Crude;
  units?: string;
  inlineSize?: number;
}

export const Value = ({
  className,
  color,
  dimensions,
  orientation,
  units = "psi",
  children,
  inlineSize = 80,
  ...props
}: ValueProps): ReactElement => {
  const borderColor = Color.cssString(color);
  const theme = Theming.use();
  const textColor: string | undefined =
    color == null
      ? "var(--pluto-gray-l0)"
      : Color.cssString(
          new Color.Color(color).pickByContrast(
            theme.colors.gray.l0,
            theme.colors.gray.l9,
          ),
        );
  const offset = (2.5 * (dimensions?.width ?? 0)) / 500;
  return (
    <Div
      className={CSS(CSS.B("value"), className)}
      {...props}
      style={{
        borderColor,
        height: dimensions?.height,
        width: "100%",
      }}
    >
      <div
        className={CSS.BE("value", "content")}
        style={{ flexGrow: 1, minWidth: dimensions?.width, inlineSize }}
      >
        {children}
      </div>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation="left" left={-2 + offset} top={50} id="1" />
        <Handle
          location="right"
          orientation="left"
          left={102 - offset}
          top={50}
          id="2"
        />
        <Handle location="top" orientation="left" left={50} top={-2} id="3" />
        <Handle location="bottom" orientation="left" left={50} top={102} id="4" />
      </HandleBoundary>
      <div className={CSS.BE("value", "units")} style={{ background: borderColor }}>
        <Text.Text level="small" color={textColor}>
          {units}
        </Text.Text>
      </div>
    </Div>
  );
};

export interface SwitchProps extends Omit<ToggleValveButtonProps, "onClick"> {
  onClick?: MouseEventHandler<HTMLInputElement>;
}

export const Switch = ({
  enabled = false,
  onClick,
  orientation = "left",
}: SwitchProps): ReactElement => (
  <Div orientation={orientation}>
    <Input.Switch value={enabled} onClick={onClick} onChange={() => {}} />
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
    </HandleBoundary>
  </Div>
);

export interface ButtonProps
  extends Omit<DivProps, "onClick">,
    Pick<CoreButton.ButtonProps, "color" | "size" | "level"> {
  label?: string;
  onClick?: MouseEventHandler<HTMLButtonElement>;
  color?: Color.Crude;
}

export const Button = ({
  onClick,
  orientation = "left",
  label = "",
  color,
  size,
  level,
}: ButtonProps): ReactElement => (
  <Div orientation={orientation}>
    <CoreButton.Button onClick={onClick} color={color} size={size} level={level}>
      {label}
    </CoreButton.Button>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={0} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={100} id="4" />
    </HandleBoundary>
  </Div>
);

export interface TextBoxProps extends DivProps, Pick<Text.TextProps, "level"> {
  text?: string;
  color?: Color.Crude;
  width?: number;
  align?: Align.Alignment;
  autoFit?: boolean;
}

export const TextBox = ({
  className,
  orientation = "left",
  text = "",
  width,
  color,
  level,
  autoFit,
  align = "center",
}: TextBoxProps): ReactElement => {
  const theme = Theming.use();
  const newcolor = Color.cssString(color ?? theme.colors.gray.l0);

  const style: CSSProperties = {
    borderColor: newcolor,
    textAlign: align as CSSProperties["textAlign"],
  };
  if (direction.construct(orientation) === "y")
    style["height"] = autoFit ? "fit-content" : width;
  else style["width"] = autoFit ? "fit-content" : width;

  return (
    <Div
      style={style}
      orientation={orientation}
      className={CSS(CSS.B("text-box"), CSS.loc(orientation), className)}
    >
      <Text.Text color={newcolor} level={level}>
        {text}
      </Text.Text>
    </Div>
  );
};

export interface SetpointProps
  extends Omit<DivProps, "onClick" | "value" | "onChange">,
    Input.Control<number> {
  dimensions?: dimensions.Dimensions;
  color?: Color.Crude;
  units?: string;
  onClick?: MouseEventHandler<HTMLButtonElement>;
}

export const Setpoint = ({
  orientation = "left",
  className,
  onClick,
  children,
  value,
  units,
  color,
  onChange,
  ...props
}: SetpointProps): ReactElement => {
  const [currValue, setCurrValue] = useState(value);
  return (
    <Div
      className={CSS(CSS.B("setpoint"), className)}
      orientation={orientation}
      {...props}
    >
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={1} top={50} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
        <Handle location="top" orientation={orientation} left={50} top={-2} id="3" />
        <Handle
          location="bottom"
          orientation={orientation}
          left={50}
          top={102}
          id="4"
        />
      </HandleBoundary>
      <Input.Numeric
        size="small"
        value={currValue}
        onChange={setCurrValue}
        showDragHandle={false}
        selectOnFocus
        endContent={units}
        color={color}
        style={{ [CSS.var("pack-border-shade")]: Color.cssString(color) }}
        borderWidth={2}
      >
        <CoreButton.Button
          variant="outlined"
          size="small"
          onClick={() => onChange(currValue)}
          color={color}
        >
          Set
        </CoreButton.Button>
      </Input.Numeric>
    </Div>
  );
};

export interface ScrewPumpProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const ScrewPump = ({
  color,
  className,
  orientation = "left",
  scale,
  ...props
}: ScrewPumpProps): ReactElement => (
  <Toggle
    {...props}
    className={CSS(CSS.B("screw-pump"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.125} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.875}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="32" cy="32" r="30" />
      <Line
        x1="32"
        y1="2"
        x2="62"
        y2="32"
        className={CSS(CSS.M("detail"), className)}
      />
      <Line
        x1="32"
        y1="62"
        x2="62"
        y2="32"
        className={CSS(CSS.M("detail"), className)}
      />
      <Path
        d="M 0 0, L -10 -10, M 0 0, L -10 10"
        transform="translate(32, 32)"
        strokeLinecap="round"
        className={CSS(CSS.M("detail"), className)}
      />
      <Path
        d="M 0 0, L -10 -10, M 0 0, L -10 10"
        transform="translate(42, 32)"
        strokeLinecap="round"
        className={CSS(CSS.M("detail"), className)}
      />
    </InternalSVG>
  </Toggle>
);

export interface VacuumPumpProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const VacuumPump = ({
  color,
  className,
  orientation = "left",
  scale,
  ...props
}: VacuumPumpProps): ReactElement => (
  <Toggle
    {...props}
    className={CSS(CSS.B("vacuum-pump"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.125} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.875}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="32" cy="32" r="30" />
      <Line
        x1="17"
        y1="6.0192"
        x2="57.9808"
        y2="17"
        className={CSS(CSS.M("detail"), className)}
      />
      <Line
        x1="17"
        y1="57.9808"
        x2="57.9808"
        y2="47"
        className={CSS(CSS.M("detail"), className)}
      />
    </InternalSVG>
  </Toggle>
);

export interface CavityPumpProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const CavityPump = ({
  color,
  className,
  orientation = "left",
  scale,
  ...props
}: CavityPumpProps): ReactElement => (
  <Toggle
    {...props}
    className={CSS(CSS.B("cavity-pump"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.125} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.875}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="32" cy="32" r="30" />
      <Line
        x1="32"
        y1="2"
        x2="62"
        y2="32"
        className={CSS(CSS.M("detail"), className)}
      />
      <Line
        x1="32"
        y1="62"
        x2="62"
        y2="32"
        className={CSS(CSS.M("detail"), className)}
      />
      <Path
        d="M 17 26 C 17 20.6667 23 20.6667 23 26 C 23 31.3333 29 31.3333 29 26 C 29 20.6667 35 20.6667 35 26"
        strokeLinecap="round"
        transform="translate(6, 6)"
        className={CSS(CSS.M("detail"), className)}
      />
    </InternalSVG>
  </Toggle>
);

export interface PistonPumpProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const PistonPump = ({
  color,
  className,
  orientation = "left",
  scale,
  ...props
}: PistonPumpProps): ReactElement => (
  <Toggle
    {...props}
    className={CSS(CSS.B("piston-pump"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.125} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.875}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="32" cy="32" r="30" />
      <Line
        x1="32"
        y1="2"
        x2="62"
        y2="32"
        className={CSS(CSS.M("detail"), className)}
      />
      <Line
        x1="32"
        y1="62"
        x2="62"
        y2="32"
        className={CSS(CSS.M("detail"), className)}
      />
      <Path
        d="M 23 32, h 16, m 0 -8, v 16"
        className={CSS(CSS.M("detail"), className)}
        strokeLinecap="round"
      />
    </InternalSVG>
  </Toggle>
);

export interface StaticMixerProps extends DivProps, SVGBasedPrimitiveProps {}

export const StaticMixer = ({
  color,
  className,
  orientation = "left",
  scale,
  ...props
}: StaticMixerProps): ReactElement => (
  <Div
    {...props}
    className={CSS(CSS.B("static-mixer"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={1.5152} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={98.4848}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 66, height: 30 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="1" y="1" width="64" height="28" rx="2" ry="2" />
      <Path d="M17 10C23 10 27 20 33 20C39 20 43 10 49 10" strokeLinecap="round" />
      <Path d="M17 20C23 20 27 10 33 10C39 10 43 20 49 20" strokeLinecap="round" />
    </InternalSVG>
  </Div>
);

export interface RotaryMixerProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const RotaryMixer = ({
  color,
  className,
  orientation = "left",
  scale,
  ...props
}: RotaryMixerProps): ReactElement => (
  <Toggle
    {...props}
    className={CSS(CSS.B("rotary-mixer"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={2} top={48.4849} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={97.5}
        top={48.4849}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 50, height: 33 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M1 30V2C1 1.44772 1.44772 1 2 1H35.4545C35.7434 1 36.0181 1.12487 36.208 1.34247L48.4262 15.3425C48.7549 15.7192 48.7549 16.2808 48.4262 16.6575L36.208 30.6575C36.0181 30.8751 35.7434 31 35.4545 31H2C1.44772 31 1 30.5523 1 30Z" />
      <Line
        x1="32"
        y1="16"
        x2="40"
        y2="16"
        strokeLinecap="round"
        className={CSS(CSS.M("detail"), className)}
      />
      <Line
        x1="32"
        y1="16"
        x2="28"
        y2="22.9282"
        strokeLinecap="round"
        className={CSS(CSS.M("detail"), className)}
      />
      <Line
        x1="32"
        y1="16"
        x2="28"
        y2="9.0717"
        strokeLinecap="round"
        className={CSS(CSS.M("detail"), className)}
      />
    </InternalSVG>
  </Toggle>
);

export interface LightProps extends DivProps, SVGBasedPrimitiveProps {
  enabled?: boolean;
}

export const Light = ({
  className,
  color,
  orientation = "left",
  enabled,
  scale,
  ...props
}: LightProps): ReactElement => (
  <Div
    {...props}
    orientation={orientation}
    className={CSS(CSS.B("light"), enabled && CSS.M("enabled"), className)}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.125} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.875}
        top={50}
        id="2"
      />
      <Handle location="top" orientation={orientation} left={50} top={3.125} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={96.75}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="32" cy="32" r="30" />
    </InternalSVG>
  </Div>
);

export interface ElectricRegulatorProps extends DivProps, SVGBasedPrimitiveProps {}

export const ElectricRegulator = ({
  className,
  orientation = "left",
  color,
  scale,
  ...props
}: ElectricRegulatorProps): ReactElement => (
  <Div className={CSS(className, CSS.B("regulator"))} {...props}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={3.4091}
        top={66.25}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7273}
        top={66.25}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 88, height: 80 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M44.5 53L7.35453 34.2035C5.35901 33.1937 3 34.6438 3 36.8803V69.1197C3 71.3562 5.35901 72.8063 7.35453 71.7965L44.5 53ZM44.5 53L81.6455 34.2035C83.641 33.1937 86 34.6438 86 36.8803V69.1197C86 71.3562 83.641 72.8063 81.6455 71.7965L44.5 53Z" />
      <Rect x="19" y="10" width="51" height="20" rx="3" ry="3" />
      <Line x1="44.5" y1="53" x2="44.5" y2="30" strokeLinecap="round" />
      <Path d="M44.5 10V8C44.5 6.34315 45.3431 5 47 5H80C81.6569 5 83 6.34315 83 8V24.4281C83 25.4126 82.517 26.3344 81.7076 26.8947L44.5 53" />
    </InternalSVG>
  </Div>
);

export interface AgitatorProps extends ToggleProps, SVGBasedPrimitiveProps {
  height?: number;
}

export const Agitator = ({
  height = 86,
  orientation = "left",
  color,
  scale,
  ...props
}: AgitatorProps): ReactElement => (
  <Toggle {...props} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary>
      <Handle location="bottom" orientation={orientation} left={50} top={100} id="1" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 86, height: height }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path
        d="M1 85V48.8541C1 46.624 3.34694 45.1735 5.34164 46.1708L80.6584 83.8292 C82.6531 84.8265 85 83.376 85 81.1459 V44"
        strokeLinecap="round"
      />
      <Path d="M43 1L43 65" strokeLinecap="round" />
    </InternalSVG>
  </Toggle>
);

export interface PropellerAgitatorProps extends AgitatorProps {}

export const PropellerAgitator = ({
  height = 86,
  orientation = "left",
  color,
  scale,
  ...props
}: PropellerAgitatorProps): ReactElement => (
  <Toggle {...props} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary>
      <Handle location="top" orientation={orientation} left={51} top={2} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 86, height: height }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M43.5 69.573L14.9534 55.6147C8.97428 52.6911 2 57.0443 2 63.6999V75.4462C2 82.1018 8.97429 86.455 14.9534 83.5314L43.5 69.573ZM43.5 69.573L72.0466 55.6147C78.0257 52.6911 85 57.0443 85 63.6999V75.4462C85 82.1018 78.0257 86.455 72.0466 83.5314L43.5 69.573Z" />
      <Path d="M43.5 69.6L43.5 2" strokeLinecap="round" />
    </InternalSVG>
  </Toggle>
);

export interface FlatBladeAgitatorProps extends AgitatorProps {}

export const FlatBladeAgitator = ({
  height = 86,
  orientation = "left",
  color,
  scale,
  ...props
}: FlatBladeAgitatorProps): ReactElement => (
  <Toggle {...props} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary>
      <Handle location="top" orientation={orientation} left={50} top={2} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 86, height: height }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1="43" y1="1" x2="43" y2="49" strokeLinecap="round" />
      <Rect x="3" y="49" width="80" height="34" rx="3" strokeLinecap="round" />
    </InternalSVG>
  </Toggle>
);

export interface PaddleAgitatorProps extends AgitatorProps {}

export const PaddleAgitator = ({
  className,
  height = 86,
  orientation = "left",
  color,
  scale,
  ...props
}: PaddleAgitatorProps): ReactElement => (
  <Toggle {...props} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary>
      <Handle location="top" orientation={orientation} left={50} top={2} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 86, height: height }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1="43" y1="1" x2="43" y2="49" strokeLinecap="round" />
      <Rect x="3" y="49" width="80" height="34" rx="3" />
      <Line
        x1="3.8"
        y1="82.1"
        x2="43"
        y2="49"
        className={CSS(CSS.M("detail"), className)}
        strokeLinecap="round"
      />
      <Line
        x1="43"
        y1="49"
        x2="43"
        y2="83"
        className={CSS(CSS.M("detail"), className)}
        strokeLinecap="round"
      />

      <Line
        x1="43"
        y1="83"
        x2="82.2"
        y2="49.9"
        className={CSS(CSS.M("detail"), className)}
        strokeLinecap="round"
      />
    </InternalSVG>
  </Toggle>
);

export interface CrossBeamAgitatorProps extends AgitatorProps {}

export const CrossBeamAgitator = ({
  className,
  height = 86,
  orientation = "left",
  color,
  scale,
  ...props
}: CrossBeamAgitatorProps): ReactElement => (
  <Toggle {...props} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary>
      <Handle location="top" orientation={orientation} left={50} top={2} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 86, height: height }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1="43" y1="1" x2="43" y2="49" strokeLinecap="round" />
      <Line x1="3" y1="49" x2="83" y2="49" strokeLinecap="round" />
      <Line x1="3" y1="83" x2="83" y2="83" strokeLinecap="round" />
      <Line x1="43" y1="49" x2="43" y2="83" strokeLinecap="round" />
      <Rect x="3" y="49" width="80" height="34" strokeWidth={0} />
    </InternalSVG>
  </Toggle>
);

export interface HelicalAgitatorProps extends AgitatorProps {}

export const HelicalAgitator = ({
  className,
  height = 86,
  orientation = "left",
  color,
  scale,
  ...props
}: HelicalAgitatorProps): ReactElement => (
  <Toggle {...props} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary>
      <Handle location="top" left={50} top={2} id="4" orientation={orientation} />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 86, height: height }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1={43} y1={1} x2={43} y2={60} strokeLinecap="round" />
      <Path
        d="M5.375 36L70.8204 48.7138C74.0584 49.3428 74.0573 53.9765 70.8189 54.6039L14.7952 65.4584C11.5729 66.0827 11.5494 70.6856 14.7651 71.3429L81.5208 84.9873"
        strokeLinecap="round"
      />
    </InternalSVG>
  </Toggle>
);

export interface CompressorProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const Compressor = ({
  color,
  className,
  orientation = "left",
  scale,
  ...props
}: CompressorProps): ReactElement => (
  <Toggle
    {...props}
    className={CSS(CSS.B("compressor"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={1.5} top={41} id="1" />
      <Handle location="right" orientation={orientation} left={95} top={3} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 68, height: 61 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="35" cy="25" r="23" />
      <Path d="M50 41.5L56.7186 50.8314C57.6711 52.1544 56.7257 54 55.0955 54H14.9045C13.2743 54 12.3289 52.1544 13.2814 50.8314L20 41.5" />
      <Line x1="35" y1="2" x2="65.0167" y2="2" />
      <Line x1="1" y1="25" x2="35" y2="25" />
    </InternalSVG>
  </Toggle>
);

export interface OffPageReferenceProps extends DivProps {
  label?: string;
  level?: Text.TextProps["level"];
  color?: Color.Crude;
}

export const OffPageReference: React.FC<OffPageReferenceProps> = ({
  id,
  className,
  orientation = "right",
  label = "text",
  color = "black",
  level = "p",
  ...props
}) => {
  const element = document.querySelector(`[data-id="${id}"]`);
  // add the orientation to the class list
  if (element) {
    element.classList.add(orientation);
  }

  const swap = direction.construct(orientation) === "y";

  return (
    <Div
      className={CSS(CSS.B("arrow"), CSS.loc(orientation), className)}
      orientation={orientation}
      {...props}
    >
      <div className="wrapper">
        <div className="outline" style={{ backgroundColor: Color.cssString(color) }}>
          <div className="bg">
            <Text.Text level={level} className={CSS.BE("symbol", "label")}>
              {label}
            </Text.Text>
          </div>
        </div>
      </div>
      <HandleBoundary orientation={orientation}>
        <Handle
          location="left"
          orientation={orientation}
          preventAutoAdjust
          left={98}
          top={50}
          swap={swap}
          id="1"
        />
        <Handle
          location="right"
          preventAutoAdjust
          orientation={orientation}
          left={1}
          top={50}
          swap={swap}
          id="2"
        />
      </HandleBoundary>
      <svg
        style={{ visibility: "hidden", position: "absolute" }}
        width="0"
        height="0"
        xmlns="http://www.w3.org/2000/svg"
        version="1.1"
      >
        <defs>
          <filter id="goo">
            <feGaussianBlur in="SourceGraphic" stdDeviation="1.5" result="blur" />
            <feColorMatrix
              in="blur"
              mode="matrix"
              values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 19 -9"
              result="goo"
            />
            <feComposite in="SourceGraphic" in2="goo" operator="atop" />
          </filter>
        </defs>
      </svg>
    </Div>
  );
};
