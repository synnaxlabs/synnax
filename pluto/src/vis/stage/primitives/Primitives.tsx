// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/stage/primitives/Primitives.css";

import { type color, type location } from "@synnaxlabs/x";
import {
  Handle as RFHandle,
  type HandleProps as RFHandleProps,
  Position as RFPosition,
  useUpdateNodeInternals,
} from "@xyflow/react";
import {
  type ComponentPropsWithoutRef,
  type MouseEventHandler,
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useRef,
} from "react";

import { Button as CoreButton } from "@/button";
import { CSS } from "@/css";

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

interface DivProps
  extends Omit<ComponentPropsWithoutRef<"div">, "color" | "onResize">,
    OrientableProps {}

const Div = ({ className, ...rest }: DivProps): ReactElement => (
  <div className={CSS(CSS.B("symbol-primitive"), className)} {...rest} />
);

export interface ButtonProps
  extends Omit<DivProps, "onClick">,
    Pick<CoreButton.ButtonProps, "color" | "size" | "level" | "onClickDelay"> {
  label?: string;
  onClick?: MouseEventHandler<HTMLButtonElement>;
  color?: color.Crude;
}

export const Button = ({
  onClick,
  orientation = "left",
  label = "",
  color,
  size,
  level,
  onClickDelay: delay,
}: ButtonProps): ReactElement => (
  <Div orientation={orientation}>
    <CoreButton.Button
      onClick={onClick}
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
