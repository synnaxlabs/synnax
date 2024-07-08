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
  type MouseEventHandler,
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useRef,
} from "react";
import {
  Handle as RFHandle,
  type HandleProps as RFHandleProps,
  Position as RFPosition,
  useUpdateNodeInternals,
} from "reactflow";

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

export interface OrientableProps {
  orientation?: location.Outer;
}

export interface SVGBasedPrimitiveProps extends OrientableProps {
  color?: Color.Crude;
  scale?: number;
}

export interface DivProps
  extends Omit<ComponentPropsWithoutRef<"div">, "color" | "onResize">,
    OrientableProps {}

interface ToggleProps
  extends Omit<ComponentPropsWithoutRef<"button">, "color" | "value"> {
  triggered?: boolean;
  enabled?: boolean;
  color?: Color.Crude;
}

interface HandleProps extends Omit<RFHandleProps, "type" | "position"> {
  orientation: location.Outer;
  location: location.Outer;
  left: number;
  top: number;
  id: string;
}

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

const adjustHandle = (
  top: number,
  left: number,
  orientation: location.Outer,
): { left: number; top: number } => {
  if (orientation === "left") return { top, left };
  if (orientation === "right") return { top: 100 - top, left: 100 - left };
  if (orientation === "top") return { top: 100 - left, left: top };
  return { top: left, left: 100 - top };
};

export interface SmartHandlesProps extends PropsWithChildren<{}>, OrientableProps {}

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

const Handle = ({
  location,
  orientation,
  left,
  top,
  ...props
}: HandleProps): ReactElement => {
  const adjusted = adjustHandle(top, left, orientation);
  return (
    <RFHandle
      position={smartPosition(location, orientation)}
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

interface ToggleValveButtonProps extends ToggleProps {
  orientation?: location.Outer;
}

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

const Div = ({ className, ...props }: DivProps): ReactElement => (
  <div className={CSS(CSS.B("symbol-primitive"), className)} {...props} />
);

export interface InternalSVGProps
  extends SVGBasedPrimitiveProps,
    Omit<ComponentPropsWithoutRef<"svg">, "direction" | "color" | "orientation"> {
  dimensions: dimensions.Dimensions;
  color?: Color.Crude;
  scale?: number;
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
  // @ts-expect-error - css variables
  if (color != null) style[CSS.var("symbol-color")] = new Color.Color(color).rgbString;
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
      <Handle location="left" orientation="left" left={0} top={50} id="1" />
      <Handle location="right" orientation="left" left={100} top={50} id="2" />
      <Handle location="top" orientation="left" left={50} top={0} id="3" />
      <Handle location="bottom" orientation="left" left={50} top={100} id="4" />
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
      <Handle location="bottom" orientation={orientation} left={50} top={100} id="1" />
      <Handle location="left" orientation={orientation} left={0} top={33} id="2" />
      <Handle location="right" orientation={orientation} left={100} top={33} id="3" />
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
      <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
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
      <Handle location="left" orientation={orientation} left={1} top={69} id="1" />
      <Handle location="right" orientation={orientation} left={99} top={69} id="2" />
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
      <Path d="M43.5 47L43.5 24.6177" />
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
      <Handle location="left" orientation={orientation} left={1} top={66} id="1" />
      <Handle location="right" orientation={orientation} left={99} top={66} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 58 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M43.5 37L6.35453 18.2035C4.35901 17.1937 2 18.6438 2 20.8803V53.1197C2 55.3562 4.35901 56.8063 6.35453 55.7965L43.5 37ZM43.5 37L80.6455 18.2035C82.641 17.1937 85 18.6438 85 20.8803V53.1197C85 55.3562 82.641 56.8063 80.6455 55.7965L43.5 37Z" />
      <Path d="M43.5 2L43.5 38" strokeLinecap="round" />
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
      <Handle location="left" orientation={orientation} left={1} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={99} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 45, height: 43 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M43 2L43 40" strokeLinecap="round" />
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
      <Handle location="bottom" orientation={orientation} left={33} top={100} id="1" />
      <Handle location="right" orientation={orientation} left={100} top={35} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M22.3611 20.1077C21.6298 20.4778 21.6298 21.5222 22.3611 21.8923L57.7433 39.7965C59.7388 40.8063 62.0978 39.3562 62.0978 37.1197L62.0978 4.88029C62.0978 2.64384 59.7388 1.19372 57.7433 2.2035L22.3611 20.1077Z" />
      <Path d="M21.8923 22.3611C21.5222 21.6298 20.4778 21.6298 20.1077 22.3611L2.20349 57.7433C1.19372 59.7388 2.64384 62.0978 4.8803 62.0978L37.1197 62.0978C39.3562 62.0978 40.8063 59.7388 39.7965 57.7433L21.8923 22.3611Z" />
      <Path d="M24.3461 18.5709L21.7484 19.88C20.9998 20.2572 20.3887 20.8601 20.0012 21.6035L19.3383 22.8756" />
    </InternalSVG>
  </Toggle>
);

export const Tag = (): ReactElement => (
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 66 25">
    <Path d="M57.0945 1.83334H4.04767C2.39087 1.83334 1.45252 3.73241 2.45908 5.0484L7.42509 11.5411C7.96869 12.2518 7.97408 13.2371 7.43828 13.9538L2.94078 19.9691C1.95471 21.2879 2.89585 23.1667 4.54256 23.1667H57.2344C57.9298 23.1667 58.5752 22.8055 58.939 22.2128L64.2505 13.5584C64.6483 12.9102 64.644 12.0925 64.2395 11.4484L58.7881 2.76955C58.4222 2.18696 57.7825 1.83334 57.0945 1.83334Z" />
  </svg>
);

export interface ReduceFittingProps extends DivProps, SVGBasedPrimitiveProps {}

export const ReduceFitting = ({
  className,
  orientation,
  color,
  scale,
  ...props
}: ReduceFittingProps): ReactElement => (
  <Div
    orientation={orientation}
    className={CSS(CSS.B("reduce-fitting"), className)}
    {...props}
  >
    <InternalSVG
      dimensions={{ width: 42, height: 43 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M38.862 31.1414L4.86205 41.3414C2.93721 41.9189 1 40.4775 1 38.4679V21.8707V4.53209C1 2.5225 2.93721 1.08116 4.86204 1.65861L38.862 11.8586C40.131 12.2393 41 13.4073 41 14.7321V21.5V28.2679C41 29.5927 40.131 30.7607 38.862 31.1414Z" />
    </InternalSVG>
  </Div>
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
      <Handle location="left" orientation={orientation} left={3} top={38.5} id="1" />
      <Handle location="right" orientation={orientation} left={95} top={2} id="2" />
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
      <Line x1="1" y1="24" x2="35" y2="24" />
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
        <Handle location="left" orientation={orientation} left={2} top={50} id="1" />
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
        <Path d="M33 24H3" strokeLinecap="round" />
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
      <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
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
      <Handle location="left" orientation={orientation} left={1} top={58} id="1" />
      <Handle location="right" orientation={orientation} left={99} top={58} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 48 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1="43.5" y1="27" x2="43.5" y2="1" />
      <Path d="M19.64 2L66.68 2" strokeLinecap="round" />
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
      <Handle location="left" orientation={orientation} left={2} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 52, height: 34 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M6 17L24.8 2.9C25.5111 2.36667 26.4889 2.36667 27.2 2.9L46 17M6 17L24.8 31.1C25.5111 31.6333 26.4889 31.6333 27.2 31.1L46 17M6 17H1M46 17H51" />
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
const DEFAULT_BORDER_RADIUS = { x: 50, y: 10 };

export interface TankProps extends DivProps {
  dimensions?: dimensions.Dimensions;
  borderRadius?: BorderRadius;
  color?: Color.Crude;
  onResize?: (dimensions: dimensions.Dimensions) => void;
}

export const Tank = ({
  className,
  dimensions = DEFAULT_DIMENSIONS,
  borderRadius = DEFAULT_BORDER_RADIUS,
  color,
  ...props
}: TankProps): ReactElement => {
  const detailedRadius = parseBorderRadius(borderRadius);
  const t = Theming.use();
  return (
    <Div
      className={CSS(className, CSS.B("tank"))}
      style={{
        ...dimensions,
        borderRadius: cssBorderRadius(detailedRadius),
        borderColor: Color.cssString(color ?? t.colors.gray.l9),
      }}
      {...props}
    >
      <HandleBoundary>
        <Handle location="top" orientation="left" left={50} top={0} id="1" />
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
        <Handle location="bottom" orientation="left" left={50} top={100} id="4" />
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
      <Handle location="left" orientation={orientation} left={0} top={66} id="1" />
      <Handle location="right" orientation={orientation} left={100} top={66} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 84, height: 58 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M42 36.5L5.41845 16.8709C3.41989 15.7985 1 17.2463 1 19.5144V53.4856C1 55.7537 3.4199 57.2015 5.41846 56.1291L42 36.5ZM42 36.5L78.5815 16.8709C80.5801 15.7985 83 17.2463 83 19.5144V53.4856C83 55.7537 80.5801 57.2015 78.5815 56.1291L42 36.5Z" />
      <Path d="M40 1.5H27.1522C24.8526 1.5 23.4081 3.9809 24.5432 5.98082L41.1303 35.2058C41.6373 36.099 43 35.7392 43 34.7122V4.5C43 2.84315 41.6569 1.5 40 1.5Z" />
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
      <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
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
      <Handle location="left" orientation={orientation} left={1} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={99} top={50} id="2" />
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
      <Handle location="bottom" orientation={orientation} left={33} top={100} id="1" />
      <Handle location="right" orientation={orientation} left={100} top={45} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 79 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M20.75 2L20.75 38" strokeLinecap="round" />
      <Path d="M9.05106 14.0802L32.4273 4.29611" strokeLinecap="round" />
      <Path d="M9.05106 20.0802L32.4273 10.2961" strokeLinecap="round" />
      <Path d="M9.05106 26.0802L32.4273 16.2961" strokeLinecap="round" />
      <Path d="M22.3611 35.1077C21.6298 35.4778 21.6298 36.5222 22.3611 36.8923L57.7433 54.7965C59.7388 55.8063 62.0978 54.3562 62.0978 52.1197L62.0978 19.8803C62.0978 17.6438 59.7388 16.1937 57.7433 17.2035L22.3611 35.1077Z" />
      <Path d="M21.8923 37.3611C21.5222 36.6298 20.4778 36.6298 20.1077 37.3611L2.20349 72.7433C1.19372 74.7388 2.64384 77.0978 4.8803 77.0978H37.1197C39.3562 77.0978 40.8063 74.7388 39.7965 72.7433L21.8923 37.3611Z" />
      <Path d="M24.3461 33.5709L21.7484 34.88C20.9998 35.2572 20.3887 35.8601 20.0012 36.6035L19.3383 37.8756" />
    </InternalSVG>
  </Div>
);

export interface ValueProps extends DivProps {
  dimensions?: dimensions.Dimensions;
  color?: Color.Crude;
  units?: string;
}

export const Value = ({
  className,
  color,
  dimensions,
  orientation,
  units = "psi",
  children,
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
        style={{ flexGrow: 1, minWidth: dimensions?.width, inlineSize: 80 }}
      >
        {children}
      </div>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation="left" left={-2} top={50} id="1" />
        <Handle location="right" orientation="left" left={102} top={50} id="2" />
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

export interface SwitchProps extends Omit<ToggleProps, "onClick">, OrientableProps {
  onClick?: MouseEventHandler<HTMLInputElement>;
}

export const Switch = ({
  enabled = false,
  onClick,
  orientation = "left",
}: SwitchProps): ReactElement => {
  return (
    <Div orientation={orientation}>
      <Input.Switch value={enabled} onClick={onClick} onChange={() => {}} />
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
      </HandleBoundary>
    </Div>
  );
};

export interface ButtonProps extends Omit<DivProps, "onClick"> {
  label?: string;
  onClick?: MouseEventHandler<HTMLButtonElement>;
}

export const Button = ({
  onClick,
  orientation = "left",
  label = "",
}: ButtonProps): ReactElement => {
  return (
    <Div orientation={orientation}>
      <CoreButton.Button onClick={onClick}>{label}</CoreButton.Button>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
        <Handle location="top" orientation={orientation} left={50} top={0} id="3" />
        <Handle
          location="bottom"
          orientation={orientation}
          left={50}
          top={100}
          id="4"
        />
      </HandleBoundary>
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
      <Handle location="left" orientation={orientation} left={1} top={40} id="1" />
      <Handle location="right" orientation={orientation} left={95} top={2.5} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 68, height: 61 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="35.1112" cy="25" r="23" />
      <Path d="M50.1112 41.5L56.8298 50.8314C57.7823 52.1544 56.8369 54 55.2067 54H15.0157C13.3855 54 12.4401 52.1544 13.3926 50.8314L20.1112 41.5" />
      <Path d="M35.1112 2H65.1279" strokeLinecap="round" />
      <Line x1="2.11121" y1="25" x2="34.1112" y2="25" strokeLinecap="round" />
      <Circle cx="35.1112" cy="37" r="11" />
      <Circle cx="35.1112" cy="13" r="11" />
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
      <Handle location="left" orientation={orientation} left={1} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 52, height: 52 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="26" cy="26" r="24" />
      <Line x1="14" y1="5.2154" x2="46.7846" y2="14" />
      <Line x1="14" y1="45.7841" x2="46.7846" y2="38" />
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
      <Handle location="left" orientation={orientation} left={1} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 52, height: 52 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="26" cy="26" r="24" />
      <Line x1="26" y1="2" x2="50" y2="26" />
      <Line x1="26" y1="50" x2="50" y2="26" />
      <Path
        d="M 17 26 C 17 20.6667 23 20.6667 23 26 C 23 31.3333 29 31.3333 29 26 C 29 20.6667 35 20.6667 35 26"
        strokeLinecap="round"
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
      <Handle location="left" orientation={orientation} left={1} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 52, height: 52 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="26" cy="26" r="24" />
      <Line x1="26" y1="2" x2="50" y2="26" />
      <Line x1="26" y1="50" x2="50" y2="26" />
      <Path d="M 18 26, L 34 26, L 34 22, M 34 26, L 34 30" strokeLinecap="round" />
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
      <Handle location="left" orientation={orientation} left={1} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={99} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 66, height: 31 }}
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
      <Handle location="left" orientation={orientation} left={1} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={99} top={48} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 50, height: 33 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M1 30V2C1 1.44772 1.44772 1 2 1H35.4545C35.7434 1 36.0181 1.12487 36.208 1.34247L48.4262 15.3425C48.7549 15.7192 48.7549 16.2808 48.4262 16.6575L36.208 30.6575C36.0181 30.8751 35.7434 31 35.4545 31H2C1.44772 31 1 30.5523 1 30Z" />
      <Line x1="32" y1="16" x2="40" y2="16" strokeLinecap="round" />
      <Line x1="32" y1="16" x2="28" y2="22.9282" strokeLinecap="round" />
      <Line x1="32" y1="16" x2="28" y2="9.0717" strokeLinecap="round" />
    </InternalSVG>
  </Toggle>
);

export interface LightProps extends ToggleProps, SVGBasedPrimitiveProps {
  // dimensions?: dimensions.Dimensions;
  color?: Color.Crude;
  // units: string;
}

export const Light = ({
  className,
  color,
  orientation = "left",
  scale,
  ...props
}: LightProps): ReactElement => (
  <Toggle {...props} className={CSS(CSS.B("light"), className)}>
    <InternalSVG
      dimensions={{ width: 50, height: 50 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="25" cy="25" r="24" />
      {/* <Circle cx="25" cy="25" r="22" fill={Color.cssString(color)} /> */}
    </InternalSVG>
  </Toggle>
);

export interface ElectricRegulatorProps extends DivProps, SVGBasedPrimitiveProps {}

export const ElectricRegulator = ({
  className,
  orientation = "left",
  color,
  scale,
  ...props
}: RegulatorProps): ReactElement => (
  <Div className={CSS(className, CSS.B("regulator"))} {...props}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={0} top={66} id="1" />
      <Handle location="right" orientation={orientation} left={100} top={66} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 84, height: 58 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M42 36.5L5.41845 16.8709C3.41989 15.7985 1 17.2463 1 19.5144V53.4856C1 55.7537 3.4199 57.2015 5.41846 56.1291L42 36.5ZM42 36.5L78.5815 16.8709C80.5801 15.7985 83 17.2463 83 19.5144V53.4856C83 55.7537 80.5801 57.2015 78.5815 56.1291L42 36.5Z" />
      <Rect width="40" height="20" x="22" y="1" />
      <Line x1="42" y1="21" x2="42" y2="35" />
      {/* <Path d="M40 1.5H27.1522C24.8526 1.5 23.4081 3.9809 24.5432 5.98082L41.1303 35.2058C41.6373 36.099 43 35.7392 43 34.7122V4.5C43 2.84315 41.6569 1.5 40 1.5Z" /> */}
    </InternalSVG>
  </Div>
);
