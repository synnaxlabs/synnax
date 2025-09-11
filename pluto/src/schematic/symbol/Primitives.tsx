// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/symbol/Primitives.css";

import { type schematic } from "@synnaxlabs/client";
import {
  color,
  dimensions,
  direction,
  type location,
  type Optional,
  xy,
} from "@synnaxlabs/x";
import {
  Handle as RFHandle,
  type HandleProps as RFHandleProps,
  Position as RFPosition,
  useUpdateNodeInternals,
} from "@xyflow/react";
import {
  type ComponentPropsWithoutRef,
  type ComponentPropsWithRef,
  type CSSProperties,
  type MouseEventHandler,
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { Button as CoreButton } from "@/button";
import { CSS } from "@/css";
import { type Flex } from "@/flex";
import { Input } from "@/input";
import { Symbol } from "@/schematic/symbol";
import { useCustom } from "@/schematic/symbol/Custom";
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

interface SmartHandlesProps extends PropsWithChildren<{}> {
  orientation: location.Outer;
  refreshDeps?: unknown;
}

const HandleBoundary = ({
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
  style,
  ...rest
}: HandleProps): ReactElement => {
  const adjusted = adjustHandle(top, left, orientation, preventAutoAdjust);
  return (
    <RFHandle
      position={swapRF(smartPosition(location, orientation), !swap)}
      {...rest}
      type="source"
      onClick={(e) => e.stopPropagation()}
      className={(CSS.B("handle"), CSS.BE("handle", rest.id))}
      style={{
        left: `${adjusted.left}%`,
        top: `${adjusted.top}%`,
        ...style,
      }}
    />
  );
};

interface ToggleProps extends Omit<ComponentPropsWithRef<"button">, "color" | "value"> {
  triggered?: boolean;
  enabled?: boolean;
  color?: color.Crude;
}

interface ToggleValveButtonProps extends ToggleProps, OrientableProps {}

const Toggle = ({
  className,
  enabled = false,
  triggered = false,
  orientation = "left",
  color: colorVal,
  ...rest
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
    color={color.cssString(colorVal)}
    {...rest}
  />
);

interface DivProps
  extends Omit<ComponentPropsWithRef<"div">, "color" | "onResize">,
    OrientableProps {}

const Div = ({ className, ...rest }: DivProps): ReactElement => (
  <div className={CSS(CSS.B("symbol-primitive"), className)} {...rest} />
);

interface SVGBasedPrimitiveProps extends OrientableProps {
  color?: color.Crude;
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

export interface FourWayValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const FourWayValve = ({
  className,
  orientation = "left",
  scale,
  color: colorVal,
  ...rest
}: FourWayValveProps): ReactElement => {
  const colorStr = color.cssString(colorVal);
  return (
    <Toggle
      {...rest}
      orientation={orientation}
      className={CSS(CSS.B("four-way-valve"), className)}
    >
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation="left" left={12.037} top={50} id="1" />
        <Handle location="right" orientation="left" left={87.963} top={50} id="2" />
        <Handle location="top" orientation="left" left={50} top={7.2916} id="3" />
        <Handle location="bottom" orientation="left" left={50} top={92.6084} id="4" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 108, height: 96 }}
        color={colorVal}
        scale={scale}
      >
        <Path
          d="M3.02937 72.7038C2.50936 73.1041 2.50936 73.8883 3.02937 74.2886L7.14001 77.453C7.79757 77.9592 8.75 77.4904 8.75 76.6606V70.3318C8.75 69.502 7.79757 69.0332 7.14001 69.5394L3.02937 72.7038Z"
          fill={colorStr}
        />
        <Path d="M54.2 48L17.0545 29.2035C15.059 28.1937 12.7 29.6438 12.7 31.8803V64.1197C12.7 66.3562 15.059 67.8063 17.0545 66.7965L54.2 48ZM54.2 48L91.3455 29.2035C93.341 28.1937 95.7 29.6438 95.7 31.8803V64.1197C95.7 66.3562 93.341 67.8063 91.3455 66.7965L54.2 48Z" />
        <Path d="M54.2 48L35.4035 85.1455C34.3937 87.141 35.8439 89.5 38.0803 89.5H70.3197C72.5562 89.5 74.0063 87.141 72.9965 85.1455L54.2 48ZM54.2 48L35.4035 10.8545C34.3937 8.85901 35.8439 6.5 38.0803 6.5H70.3197C72.5562 6.5 74.0063 8.85901 72.9965 10.8545L54.2 48Z" />
        <Path
          d="M8.70001 73.5C24.7 73.5 28.7 86.8333 28.7 93.5"
          strokeLinecap="round"
        />
        <Path
          d="M105.371 23.2962C105.891 22.8959 105.891 22.1117 105.371 21.7114L101.26 18.547C100.602 18.0408 99.65 18.5096 99.65 19.3394V25.6682C99.65 26.498 100.602 26.9668 101.26 26.4606L105.371 23.2962Z"
          fill={colorStr}
        />
        <Path d="M99.7 22.5C83.7 22.5 79.7 9.16667 79.7 2.5" strokeLinecap="round" />
      </InternalSVG>
    </Toggle>
  );
};

export interface ThreeWayValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const ThreeWayValve = ({
  color,
  orientation = "left",
  scale,
  ...rest
}: ThreeWayValveProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("three-way-valve"))} orientation={orientation}>
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
  ...rest
}: ValveProps): ReactElement => (
  <Toggle {...rest}>
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
  ...rest
}: SolenoidValveProps): ReactElement => (
  <Toggle
    className={CSS(
      CSS.B("solenoid-valve"),
      normallyOpen && CSS.M("normally-open"),
      className,
    )}
    orientation={orientation}
    {...rest}
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
      <Handle
        location="left"
        orientation={orientation}
        left={33.3333}
        top={17.7778}
        id="3"
      />
      <Handle location="top" orientation={orientation} left={50} top={2.8986} id="4" />
      <Handle
        location="right"
        orientation={orientation}
        left={66.6667}
        top={17.7778}
        id="5"
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

export interface CustomActuatorProps extends ToggleProps, SVGBasedPrimitiveProps {
  specKey: string;
  stateOverrides?: schematic.symbol.State[];
}

export const CustomActuator = ({
  specKey,
  enabled = false,
  triggered = false,
  orientation = "left",
  color: colorProp,
  scale = 1,
  className,
  stateOverrides,
  ...rest
}: CustomActuatorProps): ReactElement | null => {
  const spec = Symbol.useRetrieve({ key: specKey });
  const svgContainerRef = useRef<HTMLButtonElement>(null);
  useCustom({
    container: svgContainerRef.current,
    orientation,
    activeState: enabled ? "active" : "base",
    externalScale: scale,
    spec: spec?.data?.data,
    stateOverrides,
  });
  const handles = spec?.data?.data?.handles || [];
  return (
    <Toggle
      ref={svgContainerRef}
      className={CSS(
        CSS.BM("symbol", "custom"),
        CSS.B("custom-actuator"),
        orientation != null && CSS.loc(orientation),
        enabled && CSS.M("enabled"),
        triggered && CSS.M("triggered"),
        className,
      )}
      enabled={enabled}
      triggered={triggered}
      {...rest}
    >
      <HandleBoundary orientation={orientation}>
        {handles.map((handle) => (
          <Handle
            key={handle.key}
            id={handle.key}
            location={handle.orientation}
            orientation={orientation}
            left={handle.position.x * 100}
            top={handle.position.y * 100}
          />
        ))}
      </HandleBoundary>
    </Toggle>
  );
};

export interface CustomStaticProps extends DivProps, SVGBasedPrimitiveProps {
  specKey: string;
  stateOverrides?: schematic.symbol.State[];
}

export const CustomStatic = ({
  specKey,
  orientation = "left",
  color: colorProp,
  scale = 1,
  className,
  stateOverrides,
  ...rest
}: CustomStaticProps): ReactElement | null => {
  const spec = Symbol.useRetrieve({ key: specKey });
  const svgContainerRef = useRef<HTMLDivElement>(null);
  useCustom({
    container: svgContainerRef.current,
    orientation,
    activeState: "base",
    externalScale: scale,
    spec: spec?.data?.data,
    stateOverrides,
  });
  const handles = spec?.data?.data?.handles || [];
  return (
    <Div
      ref={svgContainerRef}
      className={CSS(
        CSS.BM("symbol", "custom"),
        CSS.B("custom-static"),
        orientation != null && CSS.loc(orientation),
        className,
      )}
      {...rest}
    >
      <HandleBoundary orientation={orientation}>
        {handles.map((handle) => (
          <Handle
            key={handle.key}
            id={handle.key}
            location={handle.orientation}
            orientation={orientation}
            left={handle.position.x * 100}
            top={handle.position.y * 100}
          />
        ))}
      </HandleBoundary>
    </Div>
  );
};

export interface ReliefValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const ReliefValve = ({
  className,
  orientation = "left",
  color,
  scale,
  enabled = false,
  ...rest
}: ReliefValveProps): ReactElement => (
  <Toggle className={CSS(CSS.B("relief-valve"), className)} enabled={enabled} {...rest}>
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
  </Toggle>
);

export interface CheckValveProps extends DivProps, SVGBasedPrimitiveProps {}

export const CheckValve = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: CheckValveProps): ReactElement => (
  <Div
    orientation={orientation}
    className={CSS(CSS.B("check-valve"), className)}
    {...rest}
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

export interface ISOCheckValveProps extends DivProps, SVGBasedPrimitiveProps {}

export const ISOCheckValve = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  ...rest
}: ISOCheckValveProps): ReactElement => {
  const colorStr = color.cssString(colorVal);
  return (
    <Div {...rest} orientation={orientation}>
      <HandleBoundary orientation={orientation}>
        <Handle
          location="left"
          orientation={orientation}
          left={8.3333}
          top={50}
          id="1"
        />
        <Handle
          location="right"
          orientation={orientation}
          left={96.4286}
          top={50}
          id="2"
        />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 84, height: 42 }}
        color={colorVal}
        orientation={orientation}
        scale={scale}
      >
        <Circle cx="7" cy="7" r="4" fill={colorStr} />
        <Path
          d="M7 39.5V11.5941C7 9.42886 9.22384 7.97669 11.2063 8.84738L76.7937 37.6526C78.7762 38.5233 81 37.0711 81 34.9059V6"
          strokeLinecap="round"
        />
      </InternalSVG>
    </Div>
  );
};

export interface CheckValveWithArrowProps extends DivProps, SVGBasedPrimitiveProps {}

export const CheckValveWithArrow = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  ...rest
}: CheckValveWithArrowProps): ReactElement => {
  const colorStr = color.cssString(colorVal);
  return (
    <Div
      orientation={orientation}
      className={CSS(CSS.B("check-valve-with-arrow"), className)}
      {...rest}
    >
      <HandleBoundary orientation={orientation}>
        <Handle
          location="left"
          orientation={orientation}
          left={8.602}
          top={60.65}
          id="1"
        />
        <Handle
          location="right"
          orientation={orientation}
          left={96.775}
          top={60.65}
          id="2"
        />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 93, height: 57 }}
        color={colorVal}
        orientation={orientation}
        scale={scale}
      >
        <Path
          d="M67.4706 5.20759C67.9906 5.6079 67.9906 6.3921 67.4706 6.79241L63.36 9.95678C62.7024 10.463 61.75 9.99421 61.75 9.16437V2.83563C61.75 2.00579 62.7024 1.53702 63.36 2.04322L67.4706 5.20759Z"
          fill={colorStr}
        />
        <Path d="M62.5 6H32.5" strokeLinecap="round" />
        <Circle cx="7.5" cy="13.5" r="6" fill={colorStr} />
        <Path d="M49 34.5L11.8545 15.7035C9.85901 14.6937 7.5 16.1438 7.5 18.3803V50.6197C7.5 52.8562 9.85901 54.3063 11.8545 53.2965L49 34.5ZM49 34.5L86.1455 15.7035C88.141 14.6937 90.5 16.1438 90.5 18.3803V50.6197C90.5 52.8562 88.141 54.3063 86.1455 53.2965L49 34.5Z" />
      </InternalSVG>
    </Div>
  );
};

export interface AngledValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const AngledValve = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: AngledValveProps): ReactElement => (
  <Toggle
    {...rest}
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

export interface BallValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const BallValve = ({
  color,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: BallValveProps): ReactElement => (
  <Toggle
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("ball-valve"), className)}
    enabled={enabled}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={2.2989} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
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
      <Circle cx="43.5978" cy="20.972" r="19" />
      <Path d="M26.5 12.8472L6.35452 2.17563C4.35901 1.16585 2 2.61598 2 4.85243V37.0918C2 39.3283 4.35901 40.7784 6.35453 39.7686L26.5 29.3472" />
      <Path d="M60.5 28.8486L80.6455 39.5202C82.641 40.5299 85 39.0798 85 36.8434V4.60396C85 2.36751 82.641 0.917381 80.6455 1.92716L60.5 12.3486" />
    </InternalSVG>
  </Toggle>
);

export interface ThreeWayBallValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const ThreeWayBallValve = ({
  color,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: ThreeWayBallValveProps): ReactElement => (
  <Toggle
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("three-way-ball-valve"), className)}
    enabled={enabled}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="bottom" orientation={orientation} left={50} top={95.8} id="1" />
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
      dimensions={{ width: 87, height: 66 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="43.5978" cy="21.722" r="19" />
      <Path d="M26.5 13.5972L6.35452 2.92563C4.35901 1.91585 2 3.36598 2 5.60243V37.8418C2 40.0783 4.35901 41.5284 6.35453 40.5186L26.5 30.0972" />
      <Path d="M60.5 29.5986L80.6455 40.2702C82.641 41.2799 85 39.8298 85 37.5934V5.35396C85 3.11751 82.641 1.66738 80.6455 2.67716L60.5 13.0986" />
      <Path d="M35.3737 38.7499L24.7021 58.8954C23.6923 60.8909 25.1425 63.2499 27.3789 63.2499H59.6183C61.8548 63.2499 63.3049 60.8909 62.2951 58.8954L51.8737 38.7499" />
    </InternalSVG>
  </Toggle>
);

export interface GateValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const GateValve = ({
  color,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: GateValveProps): ReactElement => (
  <Toggle
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("gate-valve"), className)}
    enabled={enabled}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={2.2989} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
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
      <Path d="M43.5 2 L43.5 39.7965" />
    </InternalSVG>
  </Toggle>
);

export interface ButterflyValveOneProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const ButterflyValveOne = ({
  color: colorVal,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: ButterflyValveOneProps): ReactElement => (
  <Toggle
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("butterfly-valve-one"), className)}
    enabled={enabled}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={2.2989} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 42 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M43.5 21L6.35453 2.20349C4.35901 1.19372 2 2.64384 2 4.88029V37.1197C2 39.3562 4.35901 40.8063 6.35453 39.7965L43.5 21ZM43.5 21L80.6455 2.20349C82.641 1.19372 85 2.64384 85 4.8803V37.1197C85 39.3562 82.641 40.8063 80.6455 39.7965L43.5 21Z" />
      <Path d="M43.5 2V40" />
      <Circle cx="43.5" cy="21" r="10" fill={color.cssString(colorVal)} />
    </InternalSVG>
  </Toggle>
);

export interface ButterflyValveTwoProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const ButterflyValveTwo = ({
  color: colorVal,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: ButterflyValveTwoProps): ReactElement => (
  <Toggle
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("butterfly-valve-two"), className)}
    enabled={enabled}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={2.2989} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 42 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="43.5" cy="21" r="10" fill={color.cssString(colorVal)} />
      <Rect x="2" y="2" width="83" height="38" rx="1" />
      <Path d="M2.29001 2.29004L84.7069 39.676" />
    </InternalSVG>
  </Toggle>
);

export interface BreatherValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const BreatherValve = ({
  color: colorVal,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: BreatherValveProps): ReactElement => (
  <Toggle
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("breather-valve"), className)}
    enabled={enabled}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={8.081} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={91.919}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 99, height: 57 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="91" cy="49.5" r="6" fill={color.cssString(colorVal)} />
      <Circle cx="8" cy="7.5" r="6" fill={color.cssString(colorVal)} />
      <Path d="M49.5 28.5L12.3545 9.70349C10.359 8.69372 8 10.1438 8 12.3803V44.6197C8 46.8562 10.359 48.3063 12.3545 47.2965L49.5 28.5ZM49.5 28.5L86.6455 9.70349C88.641 8.69372 91 10.1438 91 12.3803V44.6197C91 46.8562 88.641 48.3063 86.6455 47.2965L49.5 28.5Z" />
    </InternalSVG>
  </Toggle>
);

export interface PumpProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const Pump = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: PumpProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("pump"), className)} orientation={orientation}>
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
        top={96.875}
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
  color: colorVal,
  orientation = "left",
  scale,
  ...rest
}: BurstDiscProps): ReactElement => {
  const colorStr = color.cssString(colorVal);
  return (
    <Div {...rest} className={CSS(CSS.B("symbol"), className)}>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={5} top={50} id="1" />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 40, height: 48 }}
        color={colorVal}
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

export interface ISOBurstDiscProps extends DivProps, SVGBasedPrimitiveProps {}

export interface ISOBurstDiscProps extends DivProps, SVGBasedPrimitiveProps {}

export const ISOBurstDisc = ({
  className,
  color,
  orientation = "left",
  scale,
  ...rest
}: ISOBurstDiscProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("symbol"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={11.1111}
        top={50}
        id="1"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 36, height: 72 }} // Reduced to ~2/3 of original size (50x108)
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="4" y="4" width="28" height="64" rx="2" strokeWidth="2" />
      <Path
        d="M13 68V47C13 46.4477 13.4489 45.9892 13.9928 45.8933C16.1351 45.5152 21 43.7981 21 36C21 28.2019 16.1351 26.4848 13.9928 26.1068C13.4489 26.0108 13 25.5523 13 25V4"
        strokeWidth="2"
      />
    </InternalSVG>
  </Div>
);

export interface CapProps extends SVGBasedPrimitiveProps, DivProps {}

export const Cap = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: CapProps): ReactElement => (
  <Div className={CSS(CSS.B("cap"), className)} {...rest}>
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

export interface ISOCapProps extends SVGBasedPrimitiveProps, DivProps {}

export const ISOCap = ({
  className,
  orientation = "left",
  color,
  scale = 1,
  ...rest
}: ISOCapProps): ReactElement => (
  <Div className={CSS(CSS.B("cap"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={50} top={50} id="1" />
    </HandleBoundary>
    <InternalSVG
      color={color}
      dimensions={{ width: 36, height: 48 }}
      orientation={orientation}
      scale={scale * 0.6}
    >
      <Path
        d="M3 3H30C31.6569 3 33 4.34315 33 6V42C33 43.6569 31.6569 45 30 45H3"
        strokeLinecap="round"
      />
    </InternalSVG>
  </Div>
);

export interface ManualValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const ManualValve = ({
  className,
  orientation = "left",
  color,
  scale,
  enabled = false,
  ...rest
}: ManualValveProps): ReactElement => (
  <Toggle
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("manual-valve"), className)}
    enabled={enabled}
  >
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
  </Toggle>
);

export interface OrificePlateProps extends SVGBasedPrimitiveProps, DivProps {}

export const OrificePlate = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: OrificePlateProps): ReactElement => (
  <Div className={CSS(CSS.B("orifice_plate"), className)} {...rest}>
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
      color={color}
      dimensions={{ width: 72, height: 36 }}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2.5" y="2" width="67" height="31" rx="2" />
      <Path d="M24.5 2V7.5V13" strokeLinecap="round" />
      <Path d="M24.5 33V22" strokeLinecap="round" />
    </InternalSVG>
  </Div>
);

export interface FilterProps extends SVGBasedPrimitiveProps, DivProps {}

const pixelToPercent = (pixel: number, total: number): number => (pixel / total) * 100;

export const Filter = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: FilterProps): ReactElement => (
  <Div className={CSS(CSS.B("filter"), className)} {...rest}>
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
  color?: color.Crude;
  onResize?: (dimensions: dimensions.Dimensions) => void;
  boxBorderRadius?: number;
  backgroundColor?: color.Crude;
  strokeWidth?: number;
}

export const Tank = ({
  className,
  dimensions = DEFAULT_DIMENSIONS,
  borderRadius = DEFAULT_BORDER_RADIUS,
  boxBorderRadius,
  color: colorVal,
  backgroundColor,
  strokeWidth = 2,
  ...rest
}: TankProps): ReactElement => {
  const detailedRadius = parseBorderRadius(borderRadius);
  const hasCornerBoundaries = boxBorderRadius == null;
  const t = Theming.use();
  const { width, height } = dimensions;
  const refreshDeps = useMemo(
    () => [dimensions, borderRadius, detailedRadius],
    [
      detailedRadius.bottomLeft,
      detailedRadius.bottomRight,
      detailedRadius.topLeft,
      detailedRadius.topRight,
      height,
      width,
    ],
  );
  const leftOffset = pixelToPercent(1, width);
  const rightOffset = 100 - leftOffset;
  const topOffset = pixelToPercent(1, height);
  const bottomOffset = 100 - topOffset;
  return (
    <Div
      className={CSS(className, CSS.B("tank"))}
      style={{
        ...dimensions,
        borderRadius: boxBorderRadius ?? cssBorderRadius(detailedRadius),
        borderColor: color.cssString(colorVal ?? t.colors.gray.l11),
        backgroundColor: color.cssString(backgroundColor),
        borderWidth: strokeWidth,
      }}
      {...rest}
    >
      <HandleBoundary refreshDeps={refreshDeps} orientation="left">
        <Handle location="top" orientation="left" left={50} top={topOffset} id="1" />
        {hasCornerBoundaries && (
          <>
            <Handle
              location="top"
              orientation="left"
              left={leftOffset}
              top={detailedRadius.topLeft.y}
              id="2"
            />
            <Handle
              location="top"
              orientation="left"
              left={rightOffset}
              top={detailedRadius.topRight.y}
              id="3"
            />
          </>
        )}
        <Handle
          location="bottom"
          orientation="left"
          left={50}
          top={bottomOffset}
          id="4"
        />
        {hasCornerBoundaries && (
          <>
            <Handle
              location="bottom"
              orientation="left"
              left={leftOffset}
              top={100 - detailedRadius.bottomLeft.y}
              id="5"
            />
            <Handle
              location="bottom"
              orientation="left"
              left={rightOffset}
              top={100 - detailedRadius.bottomRight.y}
              id="6"
            />
          </>
        )}
        <Handle location="left" orientation="left" left={leftOffset} top={50} id="7" />
        <Handle
          location="right"
          orientation="left"
          left={rightOffset}
          top={50}
          id="8"
        />
      </HandleBoundary>
    </Div>
  );
};

export const DEFAULT_POLYGON_SIDE_LENGTH = 20;
export interface PolygonProps extends DivProps {
  numSides: number;
  sideLength: number;
  rotation?: number;
  cornerRounding?: number;
  color?: color.Crude;
  backgroundColor?: color.Crude;
  strokeWidth?: number;
}

const calculatePolygonVertices = (
  numSides: number,
  sideLength: number,
  rotationDeg: number = 0,
  padding: number = 4,
): Array<{ x: number; y: number }> => {
  const angleStep = (2 * Math.PI) / numSides;
  const rotationRad = (rotationDeg * Math.PI) / 180;
  const radius = sideLength / (2 * Math.sin(Math.PI / numSides));
  const center = radius + padding / 2;

  return Array.from({ length: numSides }).map((_, i) => {
    const angle = angleStep * i + rotationRad - Math.PI / 2;
    return {
      x: center + radius * Math.cos(angle),
      y: center + radius * Math.sin(angle),
    };
  });
};

const generateRoundedPolygonPath = (
  vertices: { x: number; y: number }[],
  sideLength: number,
  cornerRounding: number,
): string => {
  const path: string[] = [];
  const r = Math.min(cornerRounding, sideLength / 2);

  const len = vertices.length;
  if (len < 3 || r <= 0)
    return `M ${vertices.map((v) => `${v.x},${v.y}`).join(" L ")} Z`;

  for (let i = 0; i < len; i++) {
    const prev = vertices[(i - 1 + len) % len];
    const curr = vertices[i];
    const next = vertices[(i + 1) % len];

    // Direction vectors
    const dx1 = curr.x - prev.x;
    const dy1 = curr.y - prev.y;
    const dx2 = next.x - curr.x;
    const dy2 = next.y - curr.y;

    const len1 = Math.hypot(dx1, dy1);
    const len2 = Math.hypot(dx2, dy2);

    const offset1 = r / len1;
    const offset2 = r / len2;

    const p1x = curr.x - dx1 * offset1;
    const p1y = curr.y - dy1 * offset1;
    const p2x = curr.x + dx2 * offset2;
    const p2y = curr.y + dy2 * offset2;

    if (i === 0) path.push(`M ${p1x},${p1y}`);
    else path.push(`L ${p1x},${p1y}`);

    path.push(`Q ${curr.x},${curr.y} ${p2x},${p2y}`);
  }

  path.push("Z");
  return path.join(" ");
};

export const Polygon = ({
  numSides,
  sideLength,
  rotation = 0,
  color: colorVal,
  backgroundColor,
  className,
  cornerRounding,
  strokeWidth,
  ...rest
}: PolygonProps): ReactElement => {
  const theme = Theming.use();
  const padding = useMemo(() => 2 * ((strokeWidth ?? 2) + 1), [strokeWidth]);
  const vertices = useMemo(
    () => calculatePolygonVertices(numSides, sideLength, rotation, padding),
    [numSides, sideLength, rotation, padding],
  );
  const path = useMemo(
    () => generateRoundedPolygonPath(vertices, sideLength, cornerRounding ?? 0),
    [vertices, sideLength, cornerRounding],
  );
  const size = useMemo(
    () => 2 * (sideLength / (2 * Math.sin(Math.PI / numSides))) + padding,
    [sideLength, numSides, padding],
  );
  return (
    <Div className={CSS(className, CSS.B("polygon"))} {...rest}>
      <InternalSVG dimensions={{ width: size, height: size }}>
        <Path
          d={path}
          fill={color.cssString(backgroundColor ?? theme.colors.gray.l1)}
          stroke={color.cssString(colorVal ?? theme.colors.gray.l9)}
          strokeWidth={strokeWidth ?? 2}
        />
      </InternalSVG>
    </Div>
  );
};

export interface CircleShapeProps extends DivProps {
  radius: number;
  color?: color.Crude;
  backgroundColor?: color.Crude;
  strokeWidth?: number;
}

export const CircleShape = ({
  radius,
  color: colorVal,
  backgroundColor,
  className,
  strokeWidth,
  ...rest
}: CircleShapeProps): ReactElement => {
  const padding = (strokeWidth ?? 2) + 1;
  const diameter = radius * 2;
  const width = diameter + 2 * padding;
  const height = diameter + 2 * padding;
  return (
    <Div className={CSS(className, CSS.B("circle-shape"))} {...rest}>
      <HandleBoundary orientation="left" refreshDeps={radius}>
        <Handle
          location="top"
          orientation="left"
          left={50}
          top={(padding / height) * 100}
          id="1"
        />
        <Handle
          location="bottom"
          orientation="left"
          left={50}
          top={((height - padding) / height) * 100}
          id="2"
        />
        <Handle
          location="left"
          orientation="left"
          left={(padding / width) * 100}
          top={50}
          id="3"
        />
        <Handle
          location="right"
          orientation="left"
          left={((width - padding) / width) * 100}
          top={50}
          id="4"
        />
      </HandleBoundary>
      <InternalSVG dimensions={{ width, height }}>
        <Circle
          cx={width / 2}
          cy={height / 2}
          r={radius}
          stroke={color.cssString(colorVal)}
          strokeWidth={strokeWidth ?? 2}
          fill={color.cssString(backgroundColor)}
        />
      </InternalSVG>
    </Div>
  );
};

export interface RegulatorProps extends DivProps, SVGBasedPrimitiveProps {}

export const Regulator = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: RegulatorProps): ReactElement => (
  <Div className={CSS(className, CSS.B("regulator"))} {...rest}>
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
      <Handle
        location="top"
        orientation={orientation}
        left={50.9091}
        top={12.5}
        id="3"
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

export interface RegulatorManualProps extends DivProps, SVGBasedPrimitiveProps {}

export const RegulatorManual = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: RegulatorManualProps): ReactElement => (
  <Div className={CSS(className, CSS.B("regulator-manual"))} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle location="top" orientation={orientation} left={50} top={2.2989} id="1" />
      <Handle location="left" orientation={orientation} left={2.381} top={75} id="2" />
      <Handle
        location="right"
        orientation={orientation}
        left={97.619}
        top={75}
        id="3"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 84 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M43.5 20L43.5 2" strokeLinecap="round" />
      <Path d="M19.5 2L67.5 2" strokeLinecap="round" />
      <Path d="M43.5 63L6.35453 44.2035C4.35901 43.1937 2 44.6438 2 46.8803V79.1197C2 81.3562 4.35901 82.8063 6.35453 81.7965L43.5 63ZM43.5 63L80.6455 44.2035C82.641 43.1937 85 44.6438 85 46.8803V79.1197C85 81.3562 82.641 82.8063 80.6455 81.7965L43.5 63Z" />
      <Path d="M60.5 40C62.1569 40 63.5231 38.6494 63.2755 37.0111C62.641 32.8129 60.681 28.8968 57.6421 25.8579C53.8914 22.1071 48.8043 20 43.5 20C38.1957 20 33.1086 22.1071 29.3579 25.8579C26.319 28.8968 24.359 32.8129 23.7245 37.0111C23.4769 38.6494 24.8431 40 26.5 40L43.5 40H60.5Z" />
      <Path d="M60.5 40C62.1569 40 63.5231 38.6494 63.2755 37.0111C62.641 32.8129 60.681 28.8968 57.6421 25.8579C53.8914 22.1071 48.8043 20 43.5 20C38.1957 20 33.1086 22.1071 29.3579 25.8579C26.319 28.8968 24.359 32.8129 23.7245 37.0111C23.4769 38.6494 24.8431 40 26.5 40L43.5 40H60.5Z" />
      <Line x1="43.5" y1="63" x2="43.5" y2="40" />
      <Path d="M43.5 20V18C43.5 16.3431 44.8431 15 46.5 15H79.5C81.1569 15 82.5 16.3431 82.5 18V34.4281C82.5 35.4126 82.017 36.3344 81.2076 36.8947L43.5 63" />
    </InternalSVG>
  </Div>
);

export interface OrificeProps extends DivProps, SVGBasedPrimitiveProps {}

export const Orifice = ({
  className,
  orientation = "left",
  scale,
  color,
  ...rest
}: OrificeProps): ReactElement => (
  <Div className={CSS(CSS.B("orifice"), className)} {...rest}>
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

export interface NeedleValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const NeedleValve = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  enabled = false,
  ...rest
}: NeedleValveProps): ReactElement => (
  <Toggle
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("needle-valve"), className)}
    enabled={enabled}
  >
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
      color={colorVal}
      scale={scale}
    >
      <Path
        d="M43.0152 21.5391L38.237 2.62245C38.1573 2.30658 38.396 2 38.7218 2L48.2782 2C48.604 2 48.8427 2.30658 48.763 2.62245L43.9848 21.5391C43.8576 22.0425 43.1424 22.0425 43.0152 21.5391Z"
        fill={color.cssString(colorVal)}
      />
      <Path d="M43.5 21.5L6.35453 2.70349C4.35901 1.69372 2 3.14384 2 5.38029V37.6197C2 39.8562 4.35901 41.3063 6.35453 40.2965L43.5 21.5ZM43.5 21.5L80.6455 2.70349C82.641 1.69372 85 3.14384 85 5.3803V37.6197C85 39.8562 82.641 41.3063 80.6455 40.2965L43.5 21.5Z" />
    </InternalSVG>
  </Toggle>
);

export interface AngledReliefValveProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const AngledReliefValve = ({
  color,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: AngledReliefValveProps): ReactElement => (
  <Toggle
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("angled-relief-valve"), className)}
    enabled={enabled}
  >
    <HandleBoundary orientation={orientation}>
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
  </Toggle>
);

export interface ValueProps extends DivProps {
  dimensions?: dimensions.Dimensions;
  color?: color.Crude;
  units?: string;
  unitsLevel?: Text.Level;
  inlineSize?: number;
}

export const Value = ({
  className,
  color: colorVal,
  dimensions,
  orientation = "left",
  units = "psi",
  unitsLevel = "small",
  children,
  inlineSize = 80,
  ...rest
}: ValueProps): ReactElement => {
  const borderColor = color.cssString(colorVal);
  const theme = Theming.use();
  const textColor: string | undefined =
    colorVal == null
      ? "var(--pluto-gray-l0)"
      : color.cssString(
          color.pickByContrast(colorVal, theme.colors.gray.l0, theme.colors.gray.l11),
        );
  return (
    <Div
      className={CSS(CSS.B("value"), className)}
      {...rest}
      style={{
        borderColor,
        height: dimensions?.height,
      }}
    >
      <div
        className={CSS.BE("value", "content")}
        style={{
          flexGrow: 1,
          minWidth: dimensions?.width,
          inlineSize,
          maxWidth: dimensions?.width,
        }}
      >
        {children}
      </div>
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation="left" left={0} top={50} id="1" />
        <Handle location="right" orientation="left" left={100} top={50} id="2" />
        <Handle location="top" orientation="left" left={50} top={-2} id="3" />
        <Handle location="bottom" orientation="left" left={50} top={102} id="4" />
      </HandleBoundary>
      <div
        className={CSS(CSS.BE("value", "units"), CSS.M(unitsLevel))}
        style={{ background: borderColor }}
      >
        <Text.Text level={unitsLevel} color={textColor}>
          {units}
        </Text.Text>
      </div>
    </Div>
  );
};

export interface SwitchProps extends Omit<ToggleValveButtonProps, "onClick"> {
  onClick?: MouseEventHandler<HTMLElement>;
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
  extends Omit<DivProps, "onClick" | "onMouseDown" | "onMouseUp">,
    Pick<CoreButton.ButtonProps, "color" | "size" | "level" | "onClickDelay"> {
  label?: string;
  onClick?: MouseEventHandler<HTMLButtonElement>;
  onMouseDown?: MouseEventHandler<HTMLButtonElement>;
  onMouseUp?: MouseEventHandler<HTMLButtonElement>;
  color?: color.Crude;
}

export const Button = ({
  onClick,
  onMouseDown,
  onMouseUp,
  orientation = "left",
  label = "",
  color,
  size,
  level,
  onClickDelay: delay,
}: ButtonProps): ReactElement => (
  <Div orientation={orientation}>
    <CoreButton.Button
      variant="filled"
      onClick={onClick}
      onMouseDown={onMouseDown}
      onMouseUp={onMouseUp}
      color={color}
      size={size}
      level={level}
      onClickDelay={delay}
    >
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

export interface TextBoxProps
  extends Omit<DivProps, "onChange">,
    Optional<Pick<Text.EditableProps, "level" | "value" | "onChange">, "onChange"> {
  color?: color.Crude;
  width?: number;
  align?: Flex.Alignment;
  autoFit?: boolean;
}

export const TextBox = ({
  className,
  orientation = "left",
  width,
  color: colorVal,
  level,
  autoFit,
  align = "center",
  value,
  onChange,
  ...rest
}: TextBoxProps): ReactElement => {
  const divStyle: CSSProperties = {
    textAlign: align as CSSProperties["textAlign"],
  };
  if (direction.construct(orientation) === "y")
    divStyle.height = autoFit ? "fit-content" : width;
  else divStyle.width = autoFit ? "fit-content" : width;

  return (
    <Div
      style={divStyle}
      orientation={orientation}
      className={CSS(CSS.B("text-box"), CSS.loc(orientation), className)}
      {...rest}
    >
      <Text.MaybeEditable
        className={CSS.BE("symbol", "label")}
        color={color.cssString(colorVal)}
        level={level}
        value={value}
        onChange={onChange}
      />
    </Div>
  );
};

export interface SetpointProps
  extends Omit<DivProps, "onClick" | "value" | "onChange">,
    Input.Control<number>,
    Pick<Input.NumericProps, "size"> {
  dimensions?: dimensions.Dimensions;
  color?: color.Crude;
  units?: string;
  onClick?: MouseEventHandler<HTMLButtonElement>;
  disabled?: boolean;
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
  size = "small",
  disabled,
  ...rest
}: SetpointProps): ReactElement => {
  const [currValue, setCurrValue] = useState(value);
  return (
    <Div
      className={CSS(CSS.B("setpoint"), className)}
      orientation={orientation}
      {...rest}
    >
      <HandleBoundary orientation={orientation}>
        <Handle location="left" orientation={orientation} left={0.5} top={50} id="1" />
        <Handle
          location="right"
          orientation={orientation}
          left={100}
          top={50}
          id="2"
          // Filled button has a z-index of 4 so we need to set this higher to show handle above
          style={{ zIndex: 5 }}
        />
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
        size={size}
        value={currValue}
        onChange={setCurrValue}
        showDragHandle={false}
        selectOnFocus
        endContent={units}
        color={color}
        borderWidth={1}
        disabled={disabled}
      >
        <CoreButton.Button
          size={size}
          variant="filled"
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
  ...rest
}: ScrewPumpProps): ReactElement => (
  <Toggle
    {...rest}
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
      <Handle location="top" orientation={orientation} left={50} top={3.125} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={96.875}
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
  ...rest
}: VacuumPumpProps): ReactElement => (
  <Toggle
    {...rest}
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
      <Handle location="top" orientation={orientation} left={50} top={3.125} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={96.875}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M17 7.59998L59.5394 18.9984M17 58.3984L59.5394 47M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33Z" />
    </InternalSVG>
  </Toggle>
);

export interface CavityPumpProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const CavityPump = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: CavityPumpProps): ReactElement => (
  <Toggle
    {...rest}
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
      <Handle location="top" orientation={orientation} left={50} top={3.125} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={96.875}
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
  ...rest
}: PistonPumpProps): ReactElement => (
  <Toggle
    {...rest}
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
      <Handle location="top" orientation={orientation} left={50} top={3.125} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={96.875}
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
  ...rest
}: StaticMixerProps): ReactElement => (
  <Div
    {...rest}
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
      <Handle location="top" orientation={orientation} left={50} top={3.3333} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={96.6667}
        id="4"
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
  ...rest
}: RotaryMixerProps): ReactElement => (
  <Toggle
    {...rest}
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
  ...rest
}: LightProps): ReactElement => (
  <Div
    {...rest}
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
  ...rest
}: ElectricRegulatorProps): ReactElement => (
  <Div className={CSS(className, CSS.B("regulator"))} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={2.29885}
        top={69.565}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.70115}
        top={69.565}
        id="2"
      />
      <Handle
        location="left"
        orientation={orientation}
        left={23.1479}
        top={23.61}
        id="3"
      />
      <Handle location="top" orientation={orientation} left={50} top={11.11} id="4" />
      <Handle
        location="right"
        orientation={orientation}
        left={76.8521}
        top={23.61}
        id="5"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 69 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M43.5 49L6.35453 30.2035C4.35901 29.1937 2 30.6438 2 32.8803V65.1197C2 67.3562 4.35901 68.8063 6.35453 67.7965L43.5 49ZM43.5 49L80.6455 30.2035C82.641 29.1937 85 30.6438 85 32.8803V65.1197C85 67.3562 82.641 68.8063 80.6455 67.7965L43.5 49Z" />
      <Rect x="21" y="6.5" width="44" height="18" rx="2" />
      <Path d="M43 6.5V4C43 2.34315 44.3431 1 46 1H79C80.6569 1 82 2.34315 82 4V20.4281C82 21.4126 81.517 22.3344 80.7076 22.8947L43 49" />
      <Line x1="43" y1="49" x2="43" y2="24.5" />
    </InternalSVG>
  </Div>
);

export interface ElectricRegulatorMotorizedProps
  extends DivProps,
    SVGBasedPrimitiveProps {}

export const ElectricRegulatorMotorized = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: ElectricRegulatorMotorizedProps): ReactElement => (
  <Div
    className={CSS(className, CSS.B("regulator-motorized"))}
    {...rest}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={35.6321}
        top={13.72549}
        id="2"
      />
      <Handle location="top" orientation={orientation} left={50} top={1.96078} id="2" />
      <Handle
        location="right"
        orientation={orientation}
        left={64.3679}
        top={13.72549}
        id="3"
      />
      <Handle
        location="left"
        orientation={orientation}
        left={24.1379}
        top={46.078}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={75.8621}
        top={46.078}
        id="5"
      />
      <Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={78.43137}
        id="6"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={78.43137}
        id="7"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 102 }}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M43.5 80.5L6.35453 61.7035C4.35901 60.6937 2 62.1438 2 64.3803V96.6197C2 98.8562 4.35901 100.306 6.35453 99.2965L43.5 80.5ZM43.5 80.5L80.6455 61.7035C82.641 60.6937 85 62.1438 85 64.3803V96.6197C85 98.8562 82.641 100.306 80.6455 99.2965L43.5 80.5Z" />
      <Line x1="43" y1="80.5" x2="43" y2="56" />
      <Path d="M43 38V35.5C43 33.8431 44.3431 32.5 46 32.5H79C80.6569 32.5 82 33.8431 82 35.5V51.9281C82 52.9126 81.517 53.8344 80.7076 54.3947L43 80.5" />
      <Rect x="21" y="38" width="44" height="18" rx="2" />
      <Path d="M43 38V26" strokeLinecap="round" />
      <Circle cx="43" cy="14" r="12" />
    </InternalSVG>
  </Div>
);

export interface AgitatorProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const Agitator = ({
  orientation = "left",
  color,
  scale,
  ...rest
}: AgitatorProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary orientation={orientation}>
      <Handle location="top" orientation={orientation} left={50} top={1} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 88, height: 86 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M1 85V48.8541C1 46.624 3.34694 45.1735 5.34164 46.1708L80.6584 83.8292 C82.6531 84.8265 85 83.376 85 81.1459 V44" />
      <Path d="M43 1L45 65" />
    </InternalSVG>
  </Toggle>
);

export interface PropellerAgitatorProps extends AgitatorProps {}

export const PropellerAgitator = ({
  orientation = "left",
  color,
  scale,
  ...rest
}: PropellerAgitatorProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary orientation={orientation}>
      <Handle location="top" orientation={orientation} left={51} top={2} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 87 }}
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
  orientation = "left",
  color,
  scale,
  ...rest
}: FlatBladeAgitatorProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary orientation={orientation}>
      <Handle location="top" orientation={orientation} left={51} top={2} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 87 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1="43" y1="1" x2="43" y2="49" />
      <Rect x="3" y="49" width="80" height="34" rx="3" strokeLinecap="round" />
    </InternalSVG>
  </Toggle>
);

export interface PaddleAgitatorProps extends AgitatorProps {}

export const PaddleAgitator = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: PaddleAgitatorProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary orientation={orientation}>
      <Handle location="top" orientation={orientation} left={51} top={2} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 87 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1="43" y1="1" x2="43" y2="49" />
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
  orientation = "left",
  color,
  scale,
  ...rest
}: CrossBeamAgitatorProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary orientation={orientation}>
      <Handle location="top" orientation={orientation} left={51} top={2} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 87 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1="43" y1="1" x2="43" y2="49" strokeLinecap="round" />
      <Line x1="3" y1="49" x2="83" y2="49" strokeLinecap="round" />
      <Line x1="3" y1="83" x2="83" y2="83" strokeLinecap="round" />
      <Line x1="43" y1="49" x2="43" y2="83" strokeLinecap="round" />
      {/* We need this rectangle here because so that when a user hovers above to click the agitator, the rectangle changes color */}
      <Rect x="3" y="49" width="80" height="34" strokeWidth={0} />
    </InternalSVG>
  </Toggle>
);

export interface HelicalAgitatorProps extends AgitatorProps {}

export const HelicalAgitator = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: HelicalAgitatorProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary orientation={orientation}>
      <Handle location="top" orientation={orientation} left={51} top={2} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 87, height: 87 }}
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

export interface OffPageReferenceProps extends DivProps {
  label?: string;
  level?: Text.TextProps["level"];
  color?: color.Crude;
  onLabelChange?: (label: string) => void;
}

export const OffPageReference: React.FC<OffPageReferenceProps> = ({
  id,
  className,
  orientation = "right",
  label = "text",
  color: colorVal,
  level = "p",
  onLabelChange,
  ...rest
}) => {
  const element = document.querySelector(`[data-id="${id}"]`);
  // add the orientation to the class list
  if (element) element.classList.add(orientation);

  const swap = direction.construct(orientation) === "y";

  return (
    <Div
      className={CSS(CSS.B("arrow"), CSS.loc(orientation), className)}
      orientation={orientation}
      {...rest}
    >
      <div className="wrapper">
        <div className="outline" style={{ backgroundColor: color.cssString(colorVal) }}>
          <div className="bg">
            <Text.MaybeEditable
              value={label}
              onChange={onLabelChange}
              level={level}
              className={CSS.BE("symbol", "label")}
            />
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

export interface VentProps extends SVGBasedPrimitiveProps, DivProps {}

export const Vent = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: VentProps): ReactElement => (
  <Div className={CSS(CSS.B("vent"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={22.7273}
        top={50}
        id="1"
      />
      <Handle location="right" orientation={orientation} left={80} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      color={color}
      dimensions={{ width: 22, height: 32 }}
      orientation={orientation}
      scale={scale}
    >
      <Path
        d="M5 3L16.6325 13.8016C17.9107 14.9885 17.9107 17.0115 16.6325 18.1984L5 29"
        strokeLinecap="round"
      />
    </InternalSVG>
  </Div>
);

export interface ISOFilterProps extends SVGBasedPrimitiveProps, DivProps {}

export const ISOFilter = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: ISOFilterProps): ReactElement => (
  <Div className={CSS(CSS.B("iso-filter"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle location="right" orientation={orientation} left={95} top={50} id="1" />
      <Handle location="left" orientation={orientation} left={5} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 60, height: 42 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="3" y="3" width="54" height="36" rx="3" ry="3" />
      <Line x1="30" y1="3" x2="30" y2="13" strokeLinecap="round" />
      <Line x1="30" y1="17" x2="30" y2="25" strokeLinecap="round" />
      <Line x1="30" y1="29" x2="30" y2="39" strokeLinecap="round" />
    </InternalSVG>
  </Div>
);

export interface FlowStraightenerProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlowStraightener = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: FlowStraightenerProps): ReactElement => (
  <Div className={CSS(CSS.B("flow-straightener"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={97} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={93} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 72, height: 36 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2.5" y="2.5" width="67" height="31" rx="2" />
      <Path d="M10.5 26.5H60.5" strokeLinecap="round" />
      <Path d="M10.5 9.5H60.5" strokeLinecap="round" />
    </InternalSVG>
  </Div>
);

export interface HeaterElementProps extends DivProps, SVGBasedPrimitiveProps {}

export const HeaterElement = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: HeaterElementProps): ReactElement => (
  <Div className={CSS(CSS.B("heater-element"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="top"
        orientation={orientation}
        left={11.9}
        top={5.55555556}
        id="3"
      />
      <Handle
        location="bottom"
        orientation={orientation}
        left={11.9}
        top={94} // not sure why but this rounding looks better
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 21, height: 36 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M2.5 34.0005H14.5" strokeLinecap="round" />
      <Path d="M2.5 2.00049L14.5 2.00049" strokeLinecap="round" />
      <Path d="M14.3793 10.0013C14.9046 10.0013 15.4248 10.1048 15.9101 10.3058C16.3954 10.5068 16.8363 10.8015 17.2078 11.1729C17.5792 11.5443 17.8738 11.9853 18.0749 12.4706C18.2759 12.9559 18.3793 13.4761 18.3793 14.0013C18.3793 14.5266 18.2759 15.0468 18.0749 15.5321C17.8738 16.0174 17.5792 16.4583 17.2078 16.8298C16.8363 17.2012 16.3954 17.4958 15.9101 17.6969C15.4248 17.8979 14.9046 18.0013 14.3793 18.0013" />
      <Path d="M14.5 2.00049C15.0253 2.00049 15.5454 2.10395 16.0307 2.30497C16.516 2.50599 16.957 2.80063 17.3284 3.17206C17.6999 3.5435 17.9945 3.98445 18.1955 4.46975C18.3965 4.95506 18.5 5.4752 18.5 6.00049C18.5 6.52578 18.3965 7.04592 18.1955 7.53122C17.9945 8.01652 17.6999 8.45748 17.3284 8.82892C16.957 9.20035 16.516 9.49499 16.0307 9.69601C15.5454 9.89703 15.0253 10.0005 14.5 10.0005" />
      <Path d="M14.5 18.0005C15.0253 18.0005 15.5454 18.104 16.0307 18.305C16.516 18.506 16.957 18.8006 17.3284 19.1721C17.6999 19.5435 17.9945 19.9845 18.1955 20.4698C18.3965 20.9551 18.5 21.4752 18.5 22.0005C18.5 22.5258 18.3965 23.0459 18.1955 23.5312C17.9945 24.0165 17.6999 24.4575 17.3284 24.8289C16.957 25.2003 16.516 25.495 16.0307 25.696C15.5454 25.897 15.0253 26.0005 14.5 26.0005" />
      <Path d="M14.6206 25.9995C15.1459 25.9995 15.6661 26.103 16.1514 26.304C16.6367 26.505 17.0776 26.7997 17.4491 27.1711C17.8205 27.5425 18.1151 27.9835 18.3162 28.4688C18.5172 28.9541 18.6206 29.4742 18.6206 29.9995C18.6206 30.5248 18.5172 31.0449 18.3162 31.5302C18.1151 32.0155 17.8205 32.4565 17.4491 32.8279C17.0776 33.1994 16.6367 33.494 16.1514 33.695C15.6661 33.896 15.1459 33.9995 14.6206 33.9995" />
    </InternalSVG>
  </Div>
);

export interface CylinderProps extends DivProps {
  dimensions?: dimensions.Dimensions;
  borderRadius?: BorderRadius;
  color?: color.Crude;
  onResize?: (dimensions: dimensions.Dimensions) => void;
  boxBorderRadius?: number;
  backgroundColor?: color.Crude;
}

export const Cylinder = ({
  className,
  dimensions = DEFAULT_DIMENSIONS,
  borderRadius = DEFAULT_BORDER_RADIUS,
  boxBorderRadius,
  color: colorVal,
  backgroundColor,
  ...rest
}: CylinderProps): ReactElement => {
  const detailedRadius = parseBorderRadius(borderRadius);
  const t = Theming.use();
  const refreshDeps = useMemo(
    () => [dimensions, borderRadius, detailedRadius],
    [
      detailedRadius.bottomLeft,
      detailedRadius.bottomRight,
      detailedRadius.topLeft,
      detailedRadius.topRight,
      dimensions.height,
      dimensions.width,
    ],
  );
  const boardColor = color.cssString(colorVal ?? t.colors.gray.l11);
  const bgColor =
    backgroundColor == null ? undefined : color.cssString(backgroundColor);
  const widthScale = dimensions.width / 66;
  const heightScale = dimensions.height / 180;
  const transform = `scale(${widthScale},${heightScale})`;

  return (
    <Div
      className={CSS(className, CSS.B("cylinder"))}
      style={{
        ...dimensions,
      }}
      {...rest}
    >
      <svg
        width="100%"
        height="100%"
        viewBox={`0 0 ${dimensions.width} ${dimensions.height}`}
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        transform={transform}
      >
        <path
          d="M23 33.6712C11.9844 36.0332 3 42.4382 3 52.8862V174.568C3 176.225 4.34315 177.568 6 177.568H60C61.6569 177.568 63 176.225 63 174.568V52.8862C63 36.3342 40.4511 29.9292 23 33.6712ZM23 33.6712V13.3181C23 0.318109 42.9975 0.318123 42.9975 13.3181V33.6712"
          vectorEffect="non-scaling-stroke"
          strokeWidth="2"
          stroke={boardColor}
          transform={transform}
          fill={bgColor}
        />
      </svg>
      <HandleBoundary refreshDeps={refreshDeps} orientation="left">
        {/* Top  */}
        <Handle location="top" orientation="left" left={50} top={2} id="1" />
        <Handle location="left" orientation="left" left={35} top={10} id="9" />
        <Handle location="right" orientation="left" left={65} top={10} id="10" />
        {/* Bottom */}
        <Handle location="bottom" orientation="left" left={50} top={98.3333} id="2" />
        {/* Main body */}
        <Handle location="left" orientation="left" left={4} top={40} id="3" />
        <Handle location="right" orientation="left" left={96} top={40} id="4" />
        <Handle location="left" orientation="left" left={4} top={60} id="5" />
        <Handle location="right" orientation="left" left={96} top={60} id="6" />
        <Handle location="left" orientation="left" left={4} top={80} id="7" />
        <Handle location="right" orientation="left" left={96} top={80} id="8" />
      </HandleBoundary>
    </Div>
  );
};

export interface SpringLoadedReliefValveProps
  extends ToggleProps,
    SVGBasedPrimitiveProps {}

export const SpringLoadedReliefValve = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  enabled = false,
  ...rest
}: SpringLoadedReliefValveProps): ReactElement => {
  const colorStr = color.cssString(colorVal);
  return (
    <Toggle
      {...rest}
      orientation={orientation}
      className={CSS(CSS.B("spring-loaded-relief-valve"), className)}
      enabled={enabled}
    >
      <HandleBoundary orientation={orientation}>
        <Handle
          location="left"
          orientation={orientation}
          left={2.1373}
          top={69.0785}
          id="1"
        />
        <Handle
          location="right"
          orientation={orientation}
          left={96.7416}
          top={72.368}
          id="2"
        />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 89, height: 76 }}
        color={colorVal}
        orientation={orientation}
        scale={scale}
      >
        <Path
          d="M46.3625 54.1079C45.6312 54.4779 45.6311 55.5223 46.3624 55.8924L81.7435 73.7989C83.7389 74.8088 86.098 73.3588 86.0982 71.1224L86.1002 38.883C86.1003 36.6465 83.7414 35.1962 81.7458 36.2059L46.3625 54.1079Z"
          stroke={colorStr}
        />
        <Path
          d="M71 38.0014V72.0014"
          stroke={colorStr}
          strokeWidth={4}
          strokeLinecap="round"
        />
        <Path
          d="M41.6389 55.8923C42.3702 55.5222 42.3702 54.4778 41.6389 54.1077L6.2567 36.2035C4.26118 35.1937 1.90217 36.6438 1.90217 38.8803L1.90217 71.1197C1.90217 73.3562 4.26119 74.8063 6.2567 73.7965L41.6389 55.8923Z"
          stroke={colorStr}
        />
        <Circle cx="44" cy="55" r="4" fill={colorStr} />
        <Path
          d="M45 4C45 3.44772 44.5523 3 44 3C43.4477 3 43 3.44772 43 4H45ZM37.5777 24.7889L38.0249 23.8944L37.5777 24.7889ZM37.5777 21.2111L38.0249 22.1056L37.5777 21.2111ZM50.4223 31.2111L49.9751 32.1056L50.4223 31.2111ZM37.5777 44.7889L37.1305 45.6833L37.5777 44.7889ZM37.5777 41.2111L37.1305 40.3167L37.5777 41.2111ZM42.8944 47.4472L42.4472 48.3416L42.8944 47.4472ZM45 52V49.2361H43V52H45ZM43.3416 46.5528L38.0249 43.8944L37.1305 45.6833L42.4472 48.3416L43.3416 46.5528ZM38.0249 42.1056L50.8695 35.6833L49.9751 33.8944L37.1305 40.3167L38.0249 42.1056ZM50.8695 30.3167L38.0249 23.8944L37.1305 25.6833L49.9751 32.1056L50.8695 30.3167ZM38.0249 22.1056L50.8695 15.6833L49.9751 13.8944L37.1305 20.3167L38.0249 22.1056ZM50.8695 10.3167L45.5528 7.65836L44.6584 9.44721L49.9751 12.1056L50.8695 10.3167ZM45 6.76393V4H43V6.76393H45ZM45.5528 7.65836C45.214 7.48897 45 7.1427 45 6.76393H43C43 7.90025 43.642 8.93904 44.6584 9.44721L45.5528 7.65836ZM50.8695 15.6833C53.0806 14.5777 53.0807 11.4223 50.8695 10.3167L49.9751 12.1056C50.7121 12.4741 50.7121 13.5259 49.9751 13.8944L50.8695 15.6833ZM38.0249 23.8944C37.2879 23.5259 37.2879 22.4741 38.0249 22.1056L37.1305 20.3167C34.9193 21.4223 34.9194 24.5777 37.1305 25.6833L38.0249 23.8944ZM50.8695 35.6833C53.0806 34.5777 53.0807 31.4223 50.8695 30.3167L49.9751 32.1056C50.7121 32.4741 50.7121 33.5259 49.9751 33.8944L50.8695 35.6833ZM38.0249 43.8944C37.2879 43.5259 37.2879 42.4741 38.0249 42.1056L37.1305 40.3167C34.9193 41.4223 34.9194 44.5777 37.1305 45.6833L38.0249 43.8944ZM45 49.2361C45 48.0998 44.358 47.061 43.3416 46.5528L42.4472 48.3416C42.786 48.511 43 48.8573 43 49.2361H45Z"
          fill={colorStr}
          strokeWidth={1}
        />
      </InternalSVG>
    </Toggle>
  );
};

export interface AngledSpringLoadedReliefValveProps
  extends ToggleProps,
    SVGBasedPrimitiveProps {}

export const AngledSpringLoadedReliefValve = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  enabled = false,
  ...rest
}: AngledSpringLoadedReliefValveProps): ReactElement => {
  const colorStr = color.cssString(colorVal);
  return (
    <Toggle
      {...rest}
      orientation={orientation}
      className={CSS(CSS.B("spring-loaded-relief-valve"), className)}
      enabled={enabled}
    >
      <HandleBoundary orientation={orientation}>
        <Handle
          location="bottom"
          orientation={orientation}
          left={31.8182}
          top={98}
          id="1"
        />
        <Handle
          location="right"
          orientation={orientation}
          left={95.6061}
          top={55.5185}
          id="2"
        />
      </HandleBoundary>
      <InternalSVG
        dimensions={{ width: 66, height: 101 }}
        color={colorVal}
        orientation={orientation}
        scale={scale}
      >
        <Path d="M23.3625 55.6237C22.6312 55.9937 22.6311 57.0381 23.3624 57.4082L58.7435 75.3147C60.7389 76.3246 63.098 74.8747 63.0981 72.6382L63.1001 40.3988C63.1003 38.1624 60.7414 36.7121 58.7458 37.7217L23.3625 55.6237Z" />
        <Path d="M48 38.633V72.633" strokeLinecap="round" strokeWidth={4} />
        <Path d="M21.8923 58.4348C21.5222 57.7035 20.4778 57.7035 20.1077 58.4348L2.20349 93.817C1.19372 95.8125 2.64384 98.1715 4.8803 98.1715H37.1197C39.3562 98.1715 40.8063 95.8125 39.7965 93.817L21.8923 58.4348Z" />
        <Circle cx="21" cy="56.0737" r="4" fill={colorStr} />
        <Path
          d="M21 53.0105V50.0225C21 49.3397 20.6516 48.704 20.0759 48.3366L15.6419 45.507C14.4098 44.7207 14.4098 42.9214 15.6419 42.1351L26.3581 35.2965C27.5902 34.5102 27.5902 32.7109 26.3581 31.9246L15.6419 25.0859C14.4098 24.2997 14.4098 22.5003 15.6419 21.714L26.3581 14.8754C27.5902 14.0891 27.5902 12.2898 26.3581 11.5035L21.9241 8.67393C21.3484 8.30656 21 7.67087 21 6.98798V4"
          stroke={colorStr}
          strokeLinecap="round"
        />
      </InternalSVG>
    </Toggle>
  );
};

export interface TJunctionProps extends DivProps, SVGBasedPrimitiveProps {}

export const TJunction = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  ...rest
}: TJunctionProps): ReactElement => (
  <Div className={CSS(CSS.B("t-junction"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={3.8462}
        top={22.3095}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={96.1538}
        top={22.3095}
        id="2"
      />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={92.8571}
        id="3"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 39, height: 21 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Path
        d="M1.5 5.5V3.5C1.5 2.39543 2.39543 1.5 3.5 1.5H35.5C36.6046 1.5 37.5 2.39543 37.5 3.5V5.5C37.5 6.60457 36.6046 7.5 35.5 7.5H24.5C23.3954 7.5 22.5 8.39543 22.5 9.5V17.5C22.5 18.6046 21.6046 19.5 20.5 19.5H18.5C17.3954 19.5 16.5 18.6046 16.5 17.5V9.5C16.5 8.39543 15.6046 7.5 14.5 7.5H3.5C2.39543 7.5 1.5 6.60457 1.5 5.5Z"
        fill={color.cssString(colorVal)}
        stroke="none"
      />
    </InternalSVG>
  </Div>
);

export interface CrossJunctionProps extends DivProps, SVGBasedPrimitiveProps {}

export const CrossJunction = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  ...rest
}: CrossJunctionProps): ReactElement => (
  <Div className={CSS(CSS.B("t-junction"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.8462} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.1539}
        top={50}
        id="2"
      />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50.5}
        top={96.1539}
        id="3"
      />
      <Handle
        location="top"
        orientation={orientation}
        left={50.5}
        top={3.8462}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 39, height: 39 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Path
        d="M22.5 35.5C22.5 36.6046 21.6046 37.5 20.5 37.5H18.5C17.3954 37.5 16.5 36.6046 16.5 35.5V24.5C16.5 23.3954 15.6046 22.5 14.5 22.5H3.5C2.39543 22.5 1.5 21.6046 1.5 20.5V18.5C1.5 17.3954 2.39543 16.5 3.5 16.5H14.5C15.6046 16.5 16.5 15.6046 16.5 14.5V3.5C16.5 2.39543 17.3954 1.5 18.5 1.5H20.5C21.6046 1.5 22.5 2.39543 22.5 3.5V14.5C22.5 15.6046 23.3954 16.5 24.5 16.5H35.5C36.6046 16.5 37.5 17.3954 37.5 18.5V20.5C37.5 21.6046 36.6046 22.5 35.5 22.5H24.5C23.3954 22.5 22.5 23.3954 22.5 24.5V35.5Z"
        fill={color.cssString(colorVal)}
        stroke="none"
      />
    </InternalSVG>
  </Div>
);

export interface FlowmeterGeneralProps extends DivProps, SVGBasedPrimitiveProps {}

interface FlowLabelProps {
  position?: xy.XY;
  color?: color.Crude;
}

const FlowmeterLabel = ({ position, color: colorVal }: FlowLabelProps) => (
  <text
    x={position?.x ?? 57}
    y={position?.y ?? 27}
    style={{ fill: color.cssString(colorVal), fontWeight: 450 }}
    stroke="none"
  >
    F
  </text>
);

export const FlowmeterGeneral = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterGeneralProps) => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-general"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={91} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 71, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <FlowmeterLabel position={{ x: 56, y: 25 }} color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface FlowmeterElectromagneticProps
  extends DivProps,
    SVGBasedPrimitiveProps {}

export const FlowmeterElectromagnetic = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterElectromagneticProps) => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-Electromagnetic"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={91} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 71, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Path d="M47.5 17.5H55.5" strokeLinecap="round" />
      <Path d="M15.5 17.5H23.5" strokeLinecap="round" />
      <Path
        d="M23.5 17.5C23.5 16.9747 23.6035 16.4546 23.8045 15.9693C24.0055 15.484 24.3001 15.043 24.6716 14.6716C25.043 14.3001 25.484 14.0055 25.9693 13.8045C26.4546 13.6035 26.9747 13.5 27.5 13.5C28.0253 13.5 28.5454 13.6035 29.0307 13.8045C29.516 14.0055 29.957 14.3001 30.3284 14.6716C30.6999 15.043 30.9945 15.484 31.1955 15.9693C31.3965 16.4546 31.5 16.9747 31.5 17.5"
        strokeWidth="2"
      />
      <Path
        d="M31.5 17.5C31.5 16.9747 31.6035 16.4546 31.8045 15.9693C32.0055 15.484 32.3001 15.043 32.6716 14.6716C33.043 14.3001 33.484 14.0055 33.9693 13.8045C34.4546 13.6035 34.9747 13.5 35.5 13.5C36.0253 13.5 36.5454 13.6035 37.0307 13.8045C37.516 14.0055 37.957 14.3001 38.3284 14.6716C38.6999 15.043 38.9945 15.484 39.1955 15.9693C39.3965 16.4546 39.5 16.9747 39.5 17.5"
        strokeWidth="2"
      />
      <Path
        d="M39.5 17.5C39.5 16.9747 39.6035 16.4546 39.8045 15.9693C40.0055 15.484 40.3001 15.043 40.6716 14.6716C41.043 14.3001 41.484 14.0055 41.9693 13.8045C42.4546 13.6035 42.9747 13.5 43.5 13.5C44.0253 13.5 44.5454 13.6035 45.0307 13.8045C45.516 14.0055 45.957 14.3001 46.3284 14.6716C46.6999 15.043 46.9945 15.484 47.1955 15.9693C47.3965 16.4546 47.5 16.9747 47.5 17.5"
        strokeWidth="2"
      />
      <FlowmeterLabel position={{ x: 58, y: 29 }} color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface FlowmeterVariableAreaProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlowmeterVariableArea = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterVariableAreaProps) => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-VariableArea"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={91} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 71, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Path d="M46 10V25" />
      <Path d="M23 13V22" />
      <Path d="M23 13L46 10" />
      <Path d="M23 22L46 25" />
      <FlowmeterLabel color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface FlowmeterCoriolisProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlowmeterCoriolis = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterCoriolisProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-Coriolis"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={91} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 71, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Path d="M2 17.6024H28.5" strokeLinecap="round" />
      <Path d="M28.5 17.6024L34.6834 14.0324" strokeLinecap="round" />
      <Path d="M34.8 14L45.9058 20.9666" strokeLinecap="round" />
      <Path d="M51.5 17.6024L46.0141 20.8987" strokeLinecap="round" />
      <Path d="M20.5 17.6024L26.6574 14.0474" strokeLinecap="round" />
      <Path d="M26.75 14.1024L37.788 21.0265" strokeLinecap="round" />
      <Path d="M43.5 17.6024L37.8427 21.0017" strokeLinecap="round" />
      <Path d="M43.5 17.6024H69" strokeLinecap="round" />
      <FlowmeterLabel position={{ x: 58, y: 29 }} color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface FlowmeterNozzleProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlowmeterNozzle = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterNozzleProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-Nozzle"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={91} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 71, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Path d="M24 2V7V12" strokeLinecap="round" />
      <Path d="M24 12H34" strokeLinecap="round" />
      <Path d="M24 23H29H34" strokeLinecap="round" />
      <Path d="M24 33V23" strokeLinecap="round" />
      <FlowmeterLabel color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface FlowmeterVenturiProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlowmeterVenturi = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterVenturiProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-Venturi"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={98.3333}
        top={50}
        id="2"
      />
      <Handle location="top" orientation={orientation} left={50} top={5.714} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={94.386}
        id="3"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 71, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Path d="M8 33L26.5329 22.3" strokeLinecap="round" />
      <Path d="M8 2L26.5329 12.7" strokeLinecap="round" />
      <Path d="M56 33L26.5876 22.2948" strokeLinecap="round" />
      <Path d="M56 2L26.5876 12.7052" strokeLinecap="round" />
      <FlowmeterLabel position={{ x: 56, y: 29 }} color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface FlowmeterRingPistonProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlowmeterRingPiston = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterRingPistonProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-RingPiston"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={1.6667} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={5.714} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={94.386}
        id="3"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 71, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Circle cx="36.5" cy="17.5" r="10.5" strokeWidth="2" />
      <Circle cx="36.5" cy="21.5" r="6.5" strokeWidth="2" />
      <FlowmeterLabel position={{ x: 56, y: 29 }} color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface FlowmeterPositiveDisplacementProps
  extends DivProps,
    SVGBasedPrimitiveProps {}

export const FlowmeterPositiveDisplacement = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterPositiveDisplacementProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-PositiveDisplacement"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={91} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 72, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Path
        d="M41 13C41 15.4853 38.9853 17.5 36.5 17.5C34.0147 17.5 32 15.4853 32 13C32 10.5147 34.0147 8.5 36.5 8.5C38.9853 8.5 41 10.5147 41 13Z"
        strokeWidth="2"
      />
      <Path
        d="M41 22C41 24.4853 38.9853 26.5 36.5 26.5C34.0147 26.5 32 24.4853 32 22C32 19.5147 34.0147 17.5 36.5 17.5C38.9853 17.5 41 19.5147 41 22Z"
        strokeWidth="2"
      />
      <FlowmeterLabel color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface FlowmeterTurbineProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlowmeterTurbine = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterTurbineProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-Turbine"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={91} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 71, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Path d="M16.5 17.5H54.5" strokeLinecap="round" />
      <Path d="M32.5 9L35.4756 17.1753" strokeLinecap="round" />
      <Path d="M38.5 26L35.5244 17.8247" strokeLinecap="round" />
      <Path d="M32.5 26L35.4756 17.8247" strokeLinecap="round" />
      <Path d="M38.5 9L35.5244 17.1753" strokeLinecap="round" />
      <Path
        d="M32.5 9C32.5 8.20435 32.8161 7.44129 33.3787 6.87868C33.9413 6.31607 34.7044 6 35.5 6C36.2956 6 37.0587 6.31607 37.6213 6.87868C38.1839 7.44129 38.5 8.20435 38.5 9"
        strokeWidth="2"
      />
      <Path
        d="M38.5 26C38.5 26.7956 38.1839 27.5587 37.6213 28.1213C37.0587 28.6839 36.2956 29 35.5 29C34.7044 29 33.9413 28.6839 33.3787 28.1213C32.8161 27.5587 32.5 26.7956 32.5 26"
        strokeWidth="2"
      />
      <FlowmeterLabel color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface FlowmeterPulseProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlowmeterPulse = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterPulseProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-Pulse"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={91} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 71, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Path d="M31 13.5H39" strokeLinecap="round" />
      <Path d="M23 21.5H31" strokeLinecap="round" />
      <Path d="M39 21.5H47" strokeLinecap="round" />
      <Path d="M39 13.5V21.5" strokeLinecap="round" />
      <Path d="M31 13.5V21.5" strokeLinecap="round" />
      <FlowmeterLabel color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface FlowmeterFloatSensorProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlowmeterFloatSensor = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterFloatSensorProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-FloatSensor"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={91} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 71, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Path d="M25 8H46" strokeLinecap="round" />
      <Path d="M31 27H40" strokeLinecap="round" />
      <Path d="M31 27L25.046 8.11641" strokeLinecap="round" />
      <Path d="M40 27L45.954 8.11641" strokeLinecap="round" />
      <FlowmeterLabel color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface FlowmeterOrificeProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlowmeterOrifice = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: FlowmeterOrificeProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-Orifice"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={91} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 72, height: 36 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2.5" y="2" width="67" height="31" rx="2" />
      <Path d="M24.5 2V7.5V13" strokeLinecap="round" />
      <Path d="M24.5 33V22" strokeLinecap="round" />
      <FlowmeterLabel color={colorVal} />
    </InternalSVG>
  </Div>
);

export interface HeatExchangerGeneralProps extends DivProps, SVGBasedPrimitiveProps {}

export const HeatExchangerGeneral = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: HeatExchangerGeneralProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("heat-exchanger-general"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4.545} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={95.454}
        top={50}
        id="2"
      />
      <Handle location="top" orientation={orientation} left={50} top={4.545} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={95.454}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 66, height: 66 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M3 33H18.3956L33.1284 18.1508M33.4276 47.8492L48.1604 33H63M33.278 48V18" />
      <Path d="M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33L63 33Z" />
    </InternalSVG>
  </Div>
);

export interface HeatExchangerMProps extends DivProps, SVGBasedPrimitiveProps {}

export const HeatExchangerM = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: HeatExchangerMProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("heat-exchanger-M"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="right" orientation={orientation} left={89} top={27.27} id="1" />
      <Handle location="right" orientation={orientation} left={89} top={72.73} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={4.545} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={95.455}
        id="4"
      />
      <Handle location="left" orientation={orientation} left={4.545} top={50} id="5" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 66, height: 66 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33L63 33Z" />
      <Path d="M58.9698 18H23.56L38.5623 33.0023M38.5623 32.9977L23.56 48H58.9698" />
    </InternalSVG>
  </Div>
);

export interface HeatExchangerStraightTubeProps
  extends DivProps,
    SVGBasedPrimitiveProps {}

export const HeatExchangerStraightTube = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: HeatExchangerStraightTubeProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("heat-exchanger-M"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="top" orientation={orientation} left={9} top={6.25} id="1" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={24}
        top={93.75}
        id="2"
      />
      <Handle location="top" orientation={orientation} left={76} top={6.25} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={91}
        top={93.75}
        id="4"
      />
      <Handle location="left" orientation={orientation} left={1.508} top={50} id="5" />
      <Handle
        location="right"
        orientation={orientation}
        left={98.492}
        top={50}
        id="6"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 199, height: 48 }}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="3" y="3" width="193" height="42" rx="1" strokeWidth="2" />
      <Rect x="32.397" y="3" width="134.206" height="42" rx="1" strokeWidth="2" />
      <Rect x="32.397" y="13.5" width="134.206" height="21" rx="1" strokeWidth="2" />
      <Line x1="32.3769" y1="24" x2="166.623" y2="24" strokeWidth="2" />
    </InternalSVG>
  </Div>
);

export interface DiaphragmPumpProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const DiaphragmPump = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: DiaphragmPumpProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("pump"), className)} orientation={orientation}>
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
        top={96.875}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path
        d="M62 32C62 48.5685 48.5685 62 32 62M62 32C62 15.4315 48.5685 2 32 2C15.4315 2 2 15.4315 2 32C2 48.5685 15.4315 62 32 62M62 32L32 62M32 2.00269L62.0025 32.0052"
        strokeLinecap="round"
      />
      <Path
        d="M31 62C29.2204 62 27.3855 61.332 25.5927 59.9086C23.791 58.4782 22.0952 56.3316 20.6377 53.5381C19.1829 50.7496 18.0147 47.4106 17.214 43.7054C16.4137 40.0021 16 36.0237 16 32C16 27.9763 16.4137 23.9979 17.214 20.2946C18.0147 16.5895 19.1829 13.2504 20.6377 10.4619C22.0952 7.66841 23.791 5.52179 25.5927 4.09136C27.3855 2.66801 29.2204 2 31 2"
        strokeWidth="2"
      />
    </InternalSVG>
  </Toggle>
);

export interface EjectionPumpProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const EjectionPump = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: EjectionPumpProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("pump"), className)} orientation={orientation}>
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
        top={96.875}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 64 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path
        d="M62 32C62 48.5685 48.5685 62 32 62M62 32C62 15.4315 48.5685 2 32 2C15.4315 2 2 15.4315 2 32C2 48.5685 15.4315 62 32 62M62 32L32 62M32 2.00269L62.0025 32.0052"
        strokeLinecap="round"
      />
      <Path
        d="M50.3827 20.3601C47.1902 21.7605 43.4002 22.8046 39.2752 23.4077C35.1519 24.0105 30.8103 24.1557 26.5584 23.8319C22.3057 23.508 18.2597 22.7238 14.7044 21.5418C11.1449 20.3584 8.19358 18.8149 6.03181 17.0454M6.02374 46.9613C8.18198 45.1925 11.129 43.6491 14.6841 42.465C18.2349 41.2824 22.2765 40.4968 26.5256 40.1707C30.774 39.8447 35.113 39.9871 39.2353 40.5866C43.3592 41.1864 47.15 42.2268 50.3455 43.6237"
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
  ...rest
}: CompressorProps): ReactElement => (
  <Toggle
    {...rest}
    className={CSS(CSS.B("compressor"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4.55} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={95.45} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={4.55} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={95.45}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 66, height: 66 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path
        d="M17 7.59998L59.5394 18.9984M17 58.3984L59.5394 47M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33Z"
        strokeLinecap="round"
      />
    </InternalSVG>
  </Toggle>
);

export interface TurboCompressorProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const TurboCompressor = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: TurboCompressorProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("pump"), className)} orientation={orientation}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4.55} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={95.45} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={4.55} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={95.45}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 66, height: 66 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M17 7.59998L59.5394 18.9984M17 58.3984L59.5394 47M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33Z" />
      <Circle cx="33" cy="33" r="14.5" />
    </InternalSVG>
  </Toggle>
);

export interface RollerVaneCompressorProps
  extends ToggleProps,
    SVGBasedPrimitiveProps {}

export const RollerVaneCompressor = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: RollerVaneCompressorProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("pump"), className)} orientation={orientation}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4.55} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={95.45} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={4.55} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={95.45}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 66, height: 66 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M17 7.59998L59.5394 18.9984M17 58.3984L59.5394 47M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33Z" />
      <Path d="M35 12.4V20.4" />
      <Path d="M3 33H16" />
      <Path d="M35 53.6V45.6" />
    </InternalSVG>
  </Toggle>
);

export interface LiquidRingCompressorProps
  extends ToggleProps,
    SVGBasedPrimitiveProps {}

export const LiquidRingCompressor = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: LiquidRingCompressorProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("pump"), className)} orientation={orientation}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4.55} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={95.45} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={4.55} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={95.45}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 66, height: 66 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M17 7.59998L59.5394 18.9984M17 58.3984L59.5394 47M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33Z" />
      <Path d="M39 33C39 33.7879 38.8448 34.5681 38.5433 35.2961C38.2417 36.0241 37.7998 36.6855 37.2426 37.2426C36.6855 37.7998 36.0241 38.2417 35.2961 38.5433C34.5681 38.8448 33.7879 39 33 39C32.2121 39 31.4319 38.8448 30.7039 38.5433C29.9759 38.2417 29.3145 37.7998 28.7574 37.2426C28.2002 36.6855 27.7583 36.0241 27.4567 35.2961C27.1552 34.5681 27 33.7879 27 33C27 32.2121 27.1552 31.4319 27.4567 30.7039C27.7583 29.9759 28.2002 29.3145 28.7574 28.7574C29.3145 28.2002 29.9759 27.7583 30.7039 27.4567C31.4319 27.1552 32.2121 27 33 27C33.7879 27 34.5681 27.1552 35.2961 27.4567C36.0241 27.7583 36.6855 28.2002 37.2426 28.7574C37.7998 29.3145 38.2417 29.9759 38.5433 30.7039C38.8448 31.4319 39 32.2121 39 33L39 33Z" />
      <Path d="M39 33H47" />
      <Path d="M27 33H19" />
      <Path d="M36 27.804L40 20.8758" />
      <Path d="M30 27.804L26 20.8758" />
      <Path d="M30 38.196L26 45.1242" />
      <Path d="M36 38.196L40 45.1242" />
    </InternalSVG>
  </Toggle>
);

export interface EjectorCompressorProps extends ToggleProps, SVGBasedPrimitiveProps {}

export const EjectorCompressor = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: EjectorCompressorProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("pump"), className)} orientation={orientation}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4.55} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={95.45} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={4.55} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={95.45}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 66, height: 66 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M17 7.59998L59.5394 18.9984M17 58.3984L59.5394 47M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33Z" />
      <Path d="M49.1214 16.2515C43.5325 21.3055 36.2463 24.0711 28.7115 23.9986C21.1766 23.9262 13.945 21.0209 8.45428 15.8604M8.4838 50.112C13.9718 44.968 21.1918 42.0733 28.7133 42.0014C36.2348 41.9295 43.5089 44.6858 49.0942 49.7239" />
    </InternalSVG>
  </Toggle>
);

export interface CentrifugalCompressorProps
  extends ToggleProps,
    SVGBasedPrimitiveProps {}

export const CentrifugalCompressor = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: CentrifugalCompressorProps): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("pump"), className)} orientation={orientation}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4.55} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={95.45} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={4.55} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={95.45}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 66, height: 66 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M17 7.59998L59.5394 18.9984M17 58.3984L59.5394 47M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33Z" />
      <Path d="M3 33H63" />
    </InternalSVG>
  </Toggle>
);

export interface FlameArrestorProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlameArrestor = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: FlameArrestorProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flame-arrestor"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={7.575} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={92.425}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 33, height: 69 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path
        d="M16.5 2.5L16.5 66.5M2.5 34.5H30.5M2.5 18.9848H30.5M2.5 50.0152H30.5M5.3 66.5H27.7C29.2464 66.5 30.5 65.1976 30.5 63.5909V5.40909C30.5 3.80244 29.2464 2.5 27.7 2.5H5.3C3.7536 2.5 2.5 3.80245 2.5 5.40909V63.5909C2.5 65.1976 3.7536 66.5 5.3 66.5Z"
        strokeLinecap="round"
      />
    </InternalSVG>
  </Div>
);

export interface FlameArrestorDetonationProps
  extends DivProps,
    SVGBasedPrimitiveProps {}

export const FlameArrestorDetonation = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: FlameArrestorDetonationProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flame-arrestor"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.333} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.667}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 63, height: 69 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2.5" width="28" height="64" rx="3" />
      <Path d="M16 2.5L16 66.5" />
      <Path d="M2 34.5H30" />
      <Path d="M2 19H30" />
      <Path d="M2 50H30" />
      <Path d="M29.121 3.37903L61 34.5" />
      <Path d="M29.12 65.62L61 34.5" />
    </InternalSVG>
  </Div>
);

export interface FlameArrestorExplosionProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlameArrestorExplosion = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: FlameArrestorExplosionProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flame-arrestor"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.333} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.667}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 60, height: 69 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2.5" width="56" height="64" rx="3" />
      <Path d="M30 2.5L30 66.5" />
      <Path d="M16 2.5L16 66.5" />
      <Path d="M2 34.5H30" />
      <Path d="M2 19H30" />
      <Path d="M2 50H30" />
    </InternalSVG>
  </Div>
);

export interface FlameArrestorFireResProps extends DivProps, SVGBasedPrimitiveProps {}

export const FlameArrestorFireRes = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: FlameArrestorFireResProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flame-arrestor"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.333} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.667}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 63, height: 69 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2.5" width="28" height="64" rx="3" />
      <Path d="M16 2.5L16 66.5" />
      <Path d="M2 34.5H30" />
      <Path d="M2 19H30" />
      <Path d="M2 50H30" />
      <Path
        d="M29 2.5C33.2023 2.5 37.3635 3.3277 41.2459 4.93586C45.1283 6.54401 48.656 8.90111 51.6274 11.8726C54.5989 14.8441 56.956 18.3717 58.5642 22.2541C60.1723 26.1366 61 30.2977 61 34.5C61 38.7023 60.1723 42.8635 58.5642 46.7459C56.956 50.6283 54.5989 54.1559 51.6274 57.1274C48.6559 60.0989 45.1283 62.456 41.2459 64.0641C37.3635 65.6723 33.2023 66.5 29 66.5"
        strokeLinecap="square"
      />
      <Line x1="27" y1="66.5" x2="29" y2="66.5" />
      <Line x1="27" y1="2.5" x2="29" y2="2.5" />
    </InternalSVG>
  </Div>
);

export interface FlameArrestorFireResDetonationProps
  extends DivProps,
    SVGBasedPrimitiveProps {}

export const FlameArrestorFireResDetonation = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: FlameArrestorFireResDetonationProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flame-arrestor"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.333} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.667}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 63, height: 69 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2.5" width="28" height="64" rx="3" />
      <Path d="M16 2.5L16 66.5" />
      <Path d="M2 34.5H30" />
      <Path d="M2 19H30" />
      <Path d="M2 50H30" />
      <Path d="M29.121 3.37903L61 34.5" />
      <Path d="M29.12 65.62L61 34.5" />
      <Path d="M29 2.5C33.2023 2.5 37.3635 3.3277 41.2459 4.93586C45.1283 6.54401 48.656 8.90111 51.6274 11.8726C54.5989 14.8441 56.956 18.3717 58.5642 22.2541C60.1723 26.1366 61 30.2977 61 34.5C61 38.7023 60.1723 42.8635 58.5642 46.7459C56.956 50.6283 54.5989 54.1559 51.6274 57.1274C48.6559 60.0989 45.1283 62.456 41.2459 64.0641C37.3635 65.6723 33.2023 66.5 29 66.5" />
      <Line x1="27" y1="66.5" x2="29" y2="66.5" />
      <Line x1="27" y1="2.5" x2="29" y2="2.5" />
    </InternalSVG>
  </Div>
);

export interface ThrusterProps extends DivProps, SVGBasedPrimitiveProps {}

export const Thruster = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: ThrusterProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("thruster"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.125} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.875}
        top={50}
        id="2"
      />
      <Handle location="top" orientation={orientation} left={25.3} top={4.76} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={25.3}
        top={95.24}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 81, height: 42 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2.5" y="2" width="38" height="38" rx="3" />
      <Path d="M78.5 37.5117V4.51172" />
      <Path d="M40.5 11.5L76.0072 2.6232" />
      <Path d="M40.5 30.5L76.0072 39.3768" />
      <Path d="M75.6192 2.71597C75.9231 2.56695 76.2597 2.49745 76.5977 2.51399C76.9357 2.53052 77.264 2.63256 77.5518 2.81053C77.8397 2.98851 78.0776 3.23661 78.2435 3.53161C78.4093 3.82661 78.4975 4.15886 78.4999 4.49726" />
      <Path d="M78.4994 37.5101C78.4914 37.8382 78.4028 38.1592 78.2414 38.445C78.0801 38.7307 77.8509 38.9723 77.574 39.1486C77.2972 39.3248 76.9813 39.4302 76.6541 39.4555C76.3269 39.4808 75.9986 39.4252 75.698 39.2936" />
    </InternalSVG>
  </Div>
);

export interface NozzleProps extends DivProps, SVGBasedPrimitiveProps {}

export const Nozzle = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: NozzleProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("nozzle"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={18.75}
        top={17.46}
        id="1"
      />
      <Handle location="top" orientation={orientation} left={50} top={1.5873} id="2" />
      <Handle
        location="right"
        orientation={orientation}
        left={81.25}
        top={17.46}
        id="3"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 64, height: 126 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="12" y="1.6667" width="40" height="40" rx="2" />
      <Path d="M1.3333 124H62.6667" strokeLinecap="round" />
      <Path
        d="M50.1883 41.6667C41.7748 41.6667 35.0143 46.3333 42.6923 59.3333C59.3476 87.5333 63.6327 119 62.855 124"
        strokeLinecap="round"
      />
      <Path
        d="M14 41.6667C22.4135 41.6667 29.174 46.3333 21.496 59.3333C4.84066 87.5333 0.555555 119 1.33333 124"
        strokeLinecap="round"
      />
    </InternalSVG>
  </Div>
);

export interface StrainerProps extends DivProps, SVGBasedPrimitiveProps {}

export const Strainer = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: StrainerProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("strainer"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={6.06} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={93.04} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 33, height: 69 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="29" height="65" rx="1" />
      <Path d="M2.293 2.29297L29.9383 66.7986" strokeDasharray="6 6" />
    </InternalSVG>
  </Div>
);

export interface StrainerConeProps extends DivProps, SVGBasedPrimitiveProps {}

export const StrainerCone = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: StrainerConeProps): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("strainer"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={6.06} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={93.04} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={{ width: 33, height: 69 }}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="29" height="65" rx="1" />
      <Path d="M31 34.5L2.30611 2.33992" strokeDasharray="6 6" />
      <Path d="M31 34.5L2.30611 66.6601" strokeDasharray="6 6" />
    </InternalSVG>
  </Div>
);
