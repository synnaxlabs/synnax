// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  useEffect,
  type ComponentPropsWithoutRef,
  type ReactElement,
  useRef,
  type PropsWithChildren,
} from "react";

import { dimensions, type location, direction, xy } from "@synnaxlabs/x";
import {
  type HandleProps as RFHandleProps,
  Handle as RFHandle,
  Position,
  useUpdateNodeInternals,
} from "reactflow";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Theming } from "@/theming";

import "@/vis/pid/symbols/primitives/Primitives.css";

export interface OrientableProps {
  orientation?: location.Outer;
}
export type DivProps = Omit<ComponentPropsWithoutRef<"div">, "color"> &
  OrientableProps & {
    color?: Color.Crude;
  };

export interface ToggleProps
  extends Omit<ComponentPropsWithoutRef<"button">, "color" | "value">,
    OrientableProps {
  triggered?: boolean;
  enabled?: boolean;
  color?: Color.Crude;
}

export interface HandleProps extends Omit<RFHandleProps, "type" | "position"> {
  orientation: location.Outer;
  location: location.Outer;
  left: number;
  top: number;
  id: string;
}

const orientationToPosittions: Record<
  location.Outer,
  Record<location.Outer, Position>
> = {
  left: {
    left: Position.Left,
    right: Position.Right,
    top: Position.Top,
    bottom: Position.Bottom,
  },
  right: {
    left: Position.Right,
    right: Position.Left,
    top: Position.Bottom,
    bottom: Position.Top,
  },
  top: {
    left: Position.Bottom,
    right: Position.Top,
    top: Position.Left,
    bottom: Position.Right,
  },
  bottom: {
    left: Position.Top,
    right: Position.Bottom,
    top: Position.Right,
    bottom: Position.Left,
  },
};

const smartPosition = (
  position: location.Outer,
  orientation: location.Outer,
): Position => orientationToPosittions[orientation][position];

const adjustHandle = (top: number, left: number, orientation: location.Outer) => {
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
  } catch (e) {
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
    first.current = false;
    const node = ref.current.closest(".react-flow__node");
    if (node == null) return;
    const id = node.getAttribute("data-id");
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
      className={(CSS.B("handle"), CSS.BE("handle", props.id))}
      style={{
        left: `${adjusted.left}%`,
        top: `${adjusted.top}%`,
      }}
    />
  );
};

interface ToggleValveButtonProps extends ToggleProps {}

const Toggle = ({
  className,
  enabled = false,
  triggered = false,
  color,
  orientation = "left",
  ...props
}: ToggleValveButtonProps): ReactElement => {
  return (
    <button
      className={CSS(
        CSS.B("symbol"),
        CSS.B("symbol-toggle"),
        orientation != null && CSS.loc(orientation),
        enabled && CSS.M("enabled"),
        triggered && CSS.M("triggered"),
        className,
      )}
      {...props}
    />
  );
};

const Div = ({
  className,
  orientation = "left",
  color,
  ...props
}: DivProps): ReactElement => {
  return <div className={CSS(CSS.B("symbol"), className)} {...props} />;
};

export interface InternalSVGProps
  extends OrientableProps,
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
        ...dimensions.scale(dims, scale * BASE_SCALE),
        ...style,
      }}
    >
      <g overflow="hidden">{children}</g>
    </svg>
  );
};

export interface FourWayValveProps extends ToggleProps {}

export const FourWayValve = ({
  className,
  orientation,
  color,
  ...props
}: FourWayValveProps): ReactElement => {
  return (
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
      <InternalSVG dimensions={{ width: 84, height: 84 }} color={color}>
        <path d="M42 42L5.36763 23.2371C3.37136 22.2146 1 23.6643 1 25.9072V58.0928C1 60.3357 3.37136 61.7854 5.36763 60.7629L42 42ZM42 42L78.6324 23.2371C80.6286 22.2146 83 23.6643 83 25.9072V58.0928C83 60.3357 80.6286 61.7854 78.6324 60.7629L42 42Z" />
        <path d="M42 42L23.2371 78.6324C22.2146 80.6286 23.6643 83 25.9072 83H58.0928C60.3357 83 61.7854 80.6286 60.7629 78.6324L42 42ZM42 42L23.2371 5.36763C22.2146 3.37136 23.6643 1 25.9072 1H58.0928C60.3357 1 61.7854 3.37136 60.7629 5.36763L42 42Z" />
      </InternalSVG>
    </Toggle>
  );
};

export interface ThreeWayValveProps extends ToggleProps, OrientableProps {}

export const ThreeWayValve = ({
  color,
  orientation = "left",
  ...props
}: ThreeWayValveProps): ReactElement => {
  return (
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
          top={100}
          id="1"
        />
        <Handle location="left" orientation={orientation} left={0} top={33} id="2" />
        <Handle location="right" orientation={orientation} left={100} top={33} id="3" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 84, height: 64 }}
        color={color}
        orientation={orientation}
      >
        <path d="M42 20.5L5.36763 1.73708C3.37136 0.714598 1 2.16432 1 4.40721V36.5928C1 38.8357 3.37136 40.2854 5.36763 39.2629L42 20.5ZM42 20.5L78.6324 1.73708C80.6286 0.714597 83 2.16432 83 4.40721V36.5928C83 38.8357 80.6286 40.2854 78.6324 39.2629L42 20.5Z" />
        <path d="M44.6394 23.9406C43.5383 21.7131 40.3617 21.7131 39.2606 23.9406L22.3401 58.1706C21.3545 60.1645 22.8053 62.5 25.0295 62.5L58.8705 62.5C61.0947 62.5 62.5455 60.1645 61.5599 58.1706L44.6394 23.9406Z" />
      </InternalSVG>
    </Toggle>
  );
};

export interface ValveProps extends ToggleProps, OrientableProps {}

export const Valve = ({
  orientation = "left",
  color,
  ...props
}: ValveProps): ReactElement => {
  return (
    <Toggle {...props}>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 84, height: 42 }}
        color={color}
        orientation={orientation}
        overflow="hidden"
      >
        <path d="M42 21L5.41842 1.37088C3.41986 0.298479 0.999969 1.74626 0.999969 4.01436V37.9856C0.999969 40.2537 3.41986 41.7015 5.41843 40.6291L42 21ZM42 21L78.5815 1.37088C80.5801 0.29848 83 1.74626 83 4.01436V37.9856C83 40.2537 80.5801 41.7015 78.5815 40.6291L42 21Z" />
      </InternalSVG>
    </Toggle>
  );
};

export interface SolenoidValveProps extends ToggleProps, OrientableProps {
  normallyOpen?: boolean;
}

export const SolenoidValve = ({
  className,
  color,
  orientation = "left",
  normallyOpen = false,
  ...props
}: SolenoidValveProps): ReactElement => {
  return (
    <Toggle
      className={CSS(
        CSS.B("solenoid-valve"),
        normallyOpen && CSS.M("normally-open"),
        className,
      )}
      {...props}
    >
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={68} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={68} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 84, height: 66 }}
        color={color}
        orientation={orientation}
      >
        <path d="M42 45L5.41845 25.3709C3.41989 24.2985 1 25.7463 1 28.0144V61.9856C1 64.2537 3.4199 65.7015 5.41846 64.6291L42 45ZM42 45L78.5815 25.3709C80.5801 24.2985 83 25.7463 83 28.0144V61.9856C83 64.2537 80.5801 65.7015 78.5815 64.6291L42 45Z" />
        <path d="M42 45L42 23.1043" />
        <rect
          className={CSS.B("coil")}
          x="28"
          y="1"
          width="28"
          height="22"
          rx="1"
          fill={
            normallyOpen && props.enabled === false ? Color.cssString(color) : "none"
          }
        />
      </InternalSVG>
    </Toggle>
  );
};

export interface ReliefValveProps extends DivProps, OrientableProps {}

export const ReliefValve = ({
  className,
  orientation = "left",
  color,
  ...props
}: ReliefValveProps): ReactElement => {
  return (
    <Div
      orientation={orientation}
      className={CSS(CSS.B("relief-valve"), className)}
      {...props}
    >
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={66} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={66} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 84, height: 57 }}
        color={color}
        orientation={orientation}
      >
        <path d="M42 36L5.41845 16.3709C3.41989 15.2985 1 16.7463 1 19.0144V52.9856C1 55.2537 3.4199 56.7015 5.41846 55.6291L42 36ZM42 36L78.5815 16.3709C80.5801 15.2985 83 16.7463 83 19.0144V52.9856C83 55.2537 80.5801 56.7015 78.5815 55.6291L42 36Z" />
        <line x1="42" y1="-4.37114e-08" x2="42" y2="36" />
        <path d="M30.3011 12.0802L53.6773 2.29611" />
        <path d="M30.3011 18.0802L53.6773 8.29611" />
        <path d="M30.3011 24.0802L53.6773 14.2961" />
      </InternalSVG>
    </Div>
  );
};

export interface CheckValveProps extends DivProps, OrientableProps {}

export const CheckValve = ({
  className,
  orientation = "left",
  color,
  ...props
}: CheckValveProps): ReactElement => {
  return (
    <Div
      orientation={orientation}
      className={CSS(CSS.B("check-valve"), className)}
      {...props}
    >
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 43, height: 43 }}
        color={color}
        orientation={orientation}
      >
        <path d="M41 0.5L41 42.5" />
        <path d="M39.583 24.1418C41.8094 23.0385 41.8062 19.8619 39.5777 18.7631L5.33075 1.87674C3.33589 0.893122 1.00181 2.34625 1.00403 4.57043L1.03783 38.4115C1.04005 40.6357 3.37703 42.0841 5.36992 41.0965L39.583 24.1418Z" />
      </InternalSVG>
    </Div>
  );
};

export interface AngledValveProps extends ToggleProps {}

export const AngledValve = ({
  color,
  className,
  orientation = "left",
  ...props
}: AngledValveProps): ReactElement => {
  return (
    <Toggle
      {...props}
      orientation={orientation}
      className={CSS(CSS.B("angled-valve"), className)}
    >
      <HandleBoundary orientation={orientation}>
        <Handle
          location="bottom"
          orientation={orientation}
          left={33}
          top={100}
          id="1"
        />
        <Handle location="right" orientation={orientation} left={100} top={35} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 60, height: 60 }}
        color={color}
        orientation={orientation}
      >
        <path d="M20.1229 19.4889L38.909 54.6633C39.9376 56.6437 38.4854 59 36.2362 59H4.00953C1.76032 59 0.308076 56.6437 1.33675 54.6633L19.6722 19.3646C19.9651 18.8008 20.4293 18.3445 20.9981 18.0615L54.6454 1.32049C56.6441 0.326449 59 1.76507 59 3.97959V35.9901C59 38.2046 56.6441 39.6433 54.6454 38.6492L21.1247 18.497" />
      </InternalSVG>
    </Toggle>
  );
};

export const Tag = (): ReactElement => {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 66 25">
      <path d="M57.0945 1.83334H4.04767C2.39087 1.83334 1.45252 3.73241 2.45908 5.0484L7.42509 11.5411C7.96869 12.2518 7.97408 13.2371 7.43828 13.9538L2.94078 19.9691C1.95471 21.2879 2.89585 23.1667 4.54256 23.1667H57.2344C57.9298 23.1667 58.5752 22.8055 58.939 22.2128L64.2505 13.5584C64.6483 12.9102 64.644 12.0925 64.2395 11.4484L58.7881 2.76955C58.4222 2.18696 57.7825 1.83334 57.0945 1.83334Z" />
    </svg>
  );
};

export interface ReduceFittingProps extends DivProps, OrientableProps {
  color?: Color.Crude;
}

export const ReduceFitting = ({
  className,
  orientation,
  color,
  ...props
}: ReduceFittingProps): ReactElement => {
  return (
    <Div
      orientation={orientation}
      className={CSS(CSS.B("reduce-fitting"), className)}
      {...props}
    >
      <InternalSVG
        dimensions={{ width: 42, height: 43 }}
        color={color}
        orientation={orientation}
      >
        <path d="M38.862 31.1414L4.86205 41.3414C2.93721 41.9189 1 40.4775 1 38.4679V21.8707V4.53209C1 2.5225 2.93721 1.08116 4.86204 1.65861L38.862 11.8586C40.131 12.2393 41 13.4073 41 14.7321V21.5V28.2679C41 29.5927 40.131 30.7607 38.862 31.1414Z" />
      </InternalSVG>
    </Div>
  );
};

export interface PumpProps extends ToggleProps {}

export const Pump = ({
  color,
  className,
  orientation = "left",
  ...props
}: PumpProps): ReactElement => {
  const colorStr = Color.cssString(color);
  return (
    <Toggle
      {...props}
      className={CSS(CSS.B("pump"), className)}
      orientation={orientation}
    >
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation="left" left={0} top={50} id="1" />
        <Handle location="right" orientation="left" left={100} top={10} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 66, height: 60 }}
        color={color}
        orientation={orientation}
      >
        <circle cx="39" cy="30" r="23" />
        <path d="M54 46.5L60.7186 55.8314C61.6711 57.1544 60.7257 59 59.0955 59H18.9045C17.2743 59 16.3289 57.1544 17.2814 55.8314L24 46.5" />
        <path
          d="M42 30L32 24.2265V35.7735L42 30ZM0 31H33V29H0L0 31Z"
          fill={colorStr}
          stroke="none"
        />
        <path
          d="M66 7L56 1.2265V12.7735L66 7ZM38 8L57 8V6L38 6V8Z"
          fill={colorStr}
          stroke="none"
        />
      </InternalSVG>
    </Toggle>
  );
};

export interface BurstDiscProps extends DivProps, OrientableProps {
  color?: Color.Crude;
}

export const BurstDisc = ({
  className,
  color,
  orientation = "left",
  ...props
}: BurstDiscProps): ReactElement => {
  const colorStr = Color.cssString(color);
  return (
    <Div {...props} className={CSS(CSS.B("symbol"), className)}>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 42, height: 49 }}
        color={color}
        orientation={orientation}
      >
        <path
          d="M42 24.5L32 18.7265V30.2735L42 24.5ZM0.976746 25.5H33V23.5H0.976746V25.5Z"
          fill={colorStr}
        />
        <path d="M1 45.413L1 3.54787C1 1.84611 2.92941 0.863804 4.30365 1.86753C29.418 20.2107 29.0408 28.7584 4.31799 47.0846C2.94482 48.1025 1 47.1223 1 45.413Z" />
      </InternalSVG>
    </Div>
  );
};

export interface CapProps extends OrientableProps, DivProps {}

export const Cap = ({
  className,
  orientation = "left",
  color,
  ...props
}: CapProps): ReactElement => {
  return (
    <Div className={CSS(CSS.B("cap"), className)} {...props}>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
      </HandleBoundary>
      <InternalSVG
        color={color}
        dimensions={{ width: 24, height: 49 }}
        orientation={orientation}
      >
        <path d="M1 45.413L1 3.54787C1 1.84611 2.92941 0.863804 4.30365 1.86753C29.418 20.2107 29.0408 28.7584 4.31799 47.0846C2.94482 48.1025 1 47.1223 1 45.413Z" />
      </InternalSVG>
    </Div>
  );
};

export interface ManualValveProps extends OrientableProps, DivProps {}

export const ManualValve = ({
  className,
  orientation = "left",
  color,
  ...props
}: ManualValveProps): ReactElement => {
  return (
    <Div className={CSS(CSS.B("manual-valve"), className)} {...props}>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 84, height: 49 }}
        color={color}
        orientation={orientation}
      >
        <line x1="41.84" y1="27.9872" x2="41.84" y2="0.910259" />
        <path d="M17.64 1.5L64.68 1.5" />
        <path d="M42 27.5L5.41845 7.87088C3.41989 6.79848 1 8.24626 1 10.5144V44.4856C1 46.7537 3.4199 48.2015 5.41846 47.1291L42 27.5ZM42 27.5L78.5815 7.87088C80.5801 6.79848 83 8.24626 83 10.5144V44.4856C83 46.7537 80.5801 48.2015 78.5815 47.1291L42 27.5Z" />
      </InternalSVG>
    </Div>
  );
};

export interface FilterProps extends DivProps, OrientableProps {}

export const Filter = ({
  className,
  orientation = "left",
  color,
  ...props
}: FilterProps): ReactElement => {
  return (
    <Div className={CSS(CSS.B("filter"), className)} {...props}>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 80, height: 32 }}
        orientation={orientation}
        color={color}
      >
        <path d="M20 16L38.8 1.9C39.5111 1.36667 40.4889 1.36667 41.2 1.9L60 16M20 16L38.8 30.1C39.5111 30.6333 40.4889 30.6333 41.2 30.1L60 16M20 16H0M60 16H80" />
      </InternalSVG>
    </Div>
  );
};

type DetailedBorderRadius = Record<location.CornerXYString, xy.XY>;
type BorderRadius =
  | number
  | Record<direction.Direction, number>
  | Record<location.CornerXYString, number>
  | DetailedBorderRadius;

export interface TankProps extends DivProps {
  dimensions?: dimensions.Dimensions;
  borderRadius?: BorderRadius;
  color?: Color.Crude;
}

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
          left={0}
          top={detailedRadius.topLeft.y - 1}
          id="2"
        />
        <Handle
          location="top"
          orientation="left"
          left={100}
          top={detailedRadius.topRight.y - 1}
          id="3"
        />
        <Handle location="bottom" orientation="left" left={50} top={100} id="4" />
        <Handle
          location="bottom"
          orientation="left"
          left={0}
          top={100 - detailedRadius.bottomLeft.y + 1}
          id="5"
        />
        <Handle
          location="bottom"
          orientation="left"
          left={100}
          top={100 - detailedRadius.bottomRight.y + 1}
          id="6"
        />
        <Handle location="left" orientation="left" left={0} top={50} id="7" />
        <Handle location="right" orientation="left" left={100} top={50} id="8" />
      </HandleBoundary>
    </Div>
  );
};

export interface RegulatorProps extends DivProps, OrientableProps {}

export const Regulator = ({
  className,
  orientation = "left",
  color,
  ...props
}: RegulatorProps): ReactElement => {
  return (
    <Div className={CSS(className, CSS.B("regulator"))} {...props}>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={66} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={66} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 84, height: 58 }}
        orientation={orientation}
        color={color}
      >
        <path d="M42 36.5L5.41845 16.8709C3.41989 15.7985 1 17.2463 1 19.5144V53.4856C1 55.7537 3.4199 57.2015 5.41846 56.1291L42 36.5ZM42 36.5L78.5815 16.8709C80.5801 15.7985 83 17.2463 83 19.5144V53.4856C83 55.7537 80.5801 57.2015 78.5815 56.1291L42 36.5Z" />
        <path d="M40 1.5H27.1522C24.8526 1.5 23.4081 3.9809 24.5432 5.98082L41.1303 35.2058C41.6373 36.099 43 35.7392 43 34.7122V4.5C43 2.84315 41.6569 1.5 40 1.5Z" />
      </InternalSVG>
    </Div>
  );
};

export interface OrificeProps extends DivProps, OrientableProps {}

export const Orifice = ({
  className,
  orientation = "left",
  color,
  ...props
}: OrificeProps): ReactElement => {
  return (
    <Div className={CSS(CSS.B("orifice"), className)} {...props}>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 68, height: 32 }}
        orientation={orientation}
        color={color}
      >
        <path d="M1 1C26.451 19.8469 59.0915 10.5132 67 1" />
        <path d="M1 31C26.451 12.1531 59.0915 21.4868 67 31" />
      </InternalSVG>
    </Div>
  );
};

export interface NeedleValveProps extends DivProps, OrientableProps {}

export const NeedleValve = ({
  className,
  orientation = "left",
  color,
  ...props
}: NeedleValveProps): ReactElement => {
  return (
    <Div className={CSS(CSS.B("needle-valve"), className)} {...props}>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
        <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 84, height: 42.5 }}
        orientation={orientation}
        color={color}
      >
        <path
          d="M42.4856 20.9156C42.3612 21.4236 41.6388 21.4236 41.5144 20.9156L36.7267 1.36895C36.6495 1.05396 36.888 0.75 37.2123 0.75L46.7877 0.75C47.112 0.75 47.3505 1.05396 47.2733 1.36895L42.4856 20.9156Z"
          fill={Color.cssString(color)}
        />
        <path d="M42 21L5.41845 1.37088C3.41989 0.298479 1 1.74626 1 4.01436V37.9856C1 40.2537 3.4199 41.7015 5.41846 40.6291L42 21ZM42 21L78.5815 1.37088C80.5801 0.29848 83 1.74626 83 4.01436V37.9856C83 40.2537 80.5801 41.7015 78.5815 40.6291L42 21Z" />
      </InternalSVG>
    </Div>
  );
};

export interface AngledReliefValveProps extends DivProps, OrientableProps {}

export const AngledReliefValve = ({
  className,
  orientation = "left",
  color,
  ...props
}: AngledReliefValveProps): ReactElement => {
  return (
    <Div
      orientation={orientation}
      className={CSS(CSS.B("angled-relief-valve"), className)}
      {...props}
    >
      <HandleBoundary>
        <Handle
          location="bottom"
          orientation={orientation}
          left={33}
          top={100}
          id="1"
        />
        <Handle location="right" orientation={orientation} left={100} top={45} id="2" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 60, height: 76 }}
        orientation={orientation}
        color={color}
      >
        <line x1="21" y1="0.5" x2="21" y2="36.5" />
        <path d="M9.30106 12.5802L32.6773 2.79611" />
        <path d="M9.30106 18.5802L32.6773 8.79611" />
        <path d="M9.30106 24.5802L32.6773 14.7961" />
        <path d="M20.1229 34.9889L38.909 70.1633C39.9376 72.1437 38.4854 74.5 36.2362 74.5H4.00953C1.76032 74.5 0.308076 72.1437 1.33675 70.1633L19.6722 34.8646C19.9651 34.3008 20.4293 33.8445 20.9981 33.5615L54.6454 16.8205C56.6441 15.8264 59 17.2651 59 19.4796V51.4901C59 53.7046 56.6441 55.1433 54.6454 54.1492L21.1247 33.997" />
      </InternalSVG>
    </Div>
  );
};

export interface ValueProps extends DivProps {
  dimensions?: dimensions.Dimensions;
}

export const Value = ({
  className,
  color,
  dimensions,
  ...props
}: ValueProps): ReactElement => (
  <Div
    className={CSS(CSS.B("value"), className)}
    {...props}
    style={{
      borderColor: Color.cssString(color),
      ...dimensions,
    }}
  >
    <HandleBoundary>
      <Handle location="left" orientation="left" left={0} top={50} id="1" />
      <Handle location="right" orientation="left" left={100} top={50} id="2" />
      <Handle location="top" orientation="left" left={50} top={0} id="3" />
      <Handle location="bottom" orientation="left" left={50} top={100} id="4" />
    </HandleBoundary>
    {props.children}
  </Div>
);
