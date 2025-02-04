// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/legend/Container.css";

import { box, location, scale, xy } from "@synnaxlabs/x";
import {
  type CSSProperties,
  memo,
  type ReactElement,
  type RefObject,
  useCallback,
  useRef,
} from "react";
import { z } from "zod";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { type OptionalControl } from "@/input/types";
import { state } from "@/state";
import { preventDefault } from "@/util/event";

export const completeStickyXYz = xy.xy.extend({
  root: location.corner,
  units: z.object({
    x: z.enum(["px", "decimal"]),
    y: z.enum(["px", "decimal"]),
  }),
});
export type CompleteStickyXY = z.infer<typeof completeStickyXYz>;
export const stickyXYz = completeStickyXYz.partial({
  root: true,
  units: true,
});

// StickyXY is a special coordinate system that allows for an element to 'stick' to an
// edge of its parent container. This makes for intuitive positioning behavior for
// components like legends. When the parent resizes, the legend will remain in
// a natural position.
export type StickyXY = z.infer<typeof stickyXYz>;

const stickyToCSS = (pos: StickyXY): CSSProperties => {
  const ret: CSSProperties = {};
  ret[pos.root?.x ?? "left"] =
    pos?.units?.x === "px" ? `${pos.x}px` : `${pos.x * 100}%`;
  ret[pos.root?.y ?? "top"] = pos?.units?.y === "px" ? `${pos.y}px` : `${pos.y * 100}%`;
  return ret;
};

/**
 * Converts a StickyXY position to a decimal position relative to the parent container
 * and correctly offset based on the dimensions of the child.
 * @param pos - The StickyXY position to convert
 * @param ref - The ref of the element being positioned. The parent will be inferred from
 * the parentElement of the ref.
 */
const stickyToDecimalXY = (
  pos: StickyXY,
  ref: RefObject<HTMLDivElement | null>,
): xy.XY => {
  const ret = { x: pos.x, y: pos.y };
  if (ref.current == null) return ret;
  const b = box.construct(ref.current);
  const parentBox = box.construct(ref.current.parentElement as HTMLDivElement);
  if (pos.units?.x === "decimal") {
    if (pos.root?.x === "right") ret.x = 1 - pos.x;
  } else if (pos.root?.x === "right")
    ret.x = 1 - (pos.x + box.width(b)) / box.width(parentBox);
  else ret.x /= box.width(parentBox);
  if (pos.units?.y === "decimal") {
    if (pos.root?.y === "bottom") ret.y = 1 - pos.y;
  } else if (pos.root?.y === "bottom")
    ret.y = 1 - (pos.y + box.height(b)) / box.height(parentBox);
  else ret.y /= box.height(parentBox);
  return ret;
};

export const calcStickyPos = (
  pos: xy.XY,
  ref: RefObject<HTMLDivElement | null>,
): StickyXY | null => {
  if (ref.current == null) return null;
  const parentBox = box.construct(ref.current.parentElement as HTMLDivElement);
  const b = box.construct(ref.current);
  const ret: CompleteStickyXY = {
    x: pos.x,
    y: pos.y,
    root: { ...location.TOP_LEFT },
    units: { x: "px", y: "px" },
  };
  if (pos.x > 0.8) {
    ret.x = (1 - pos.x) * box.width(parentBox) - box.width(b);
    ret.root.x = "right";
  } else if (pos.x < 0.2) ret.x = pos.x * box.width(parentBox);
  else ret.units.x = "decimal";
  if (pos.y > 0.8) {
    ret.y = (1 - pos.y) * box.height(parentBox) - box.height(b);
    ret.root.y = "bottom";
  } else if (pos.y < 0.2) ret.y = pos.y * box.height(parentBox);
  else ret.units.y = "decimal";
  ret.x = Math.round(ret.x * 100) / 100;
  return { ...ret, ...xy.truncate(ret, 3) };
};

export interface ContainerProps
  extends Omit<Align.SpaceProps, "onChange">,
    Partial<OptionalControl<StickyXY>> {
  dragEnabled?: boolean;
}

const TOP_LEFT_DECIMAL = box.reRoot(box.DECIMAL, location.TOP_LEFT);

export const Container = memo(
  ({
    className,
    value,
    onChange,
    style,
    draggable = true,
    ...props
  }: ContainerProps): ReactElement | null => {
    const [position, setPosition] = state.usePurePassthrough<StickyXY>({
      value,
      onChange,
      initial: {
        x: 0.1,
        y: 0.1,
        root: location.TOP_LEFT,
        units: { x: "decimal", y: "decimal" },
      },
    });

    const positionRef = useRef<StickyXY>(position);
    const disabled = useSyncedRef(draggable === false);
    const ref = useRef<HTMLDivElement | null>(null);

    if (position !== null) style = { ...style, ...stickyToCSS(position) };

    const calculatePosition = useCallback((drag: box.Box): StickyXY | null => {
      if (ref.current?.parentElement == null) return positionRef.current;
      const bounds = box.construct(ref.current.parentElement);
      const decimalScale = scale.XY.scale(bounds).scale(TOP_LEFT_DECIMAL);
      const elDecimal = decimalScale.box(box.construct(ref.current));
      // Clamp the position to the bounds of the parent --along-- with the dimensions
      // of the element being dragged. This prevents the right and bottom edges from
      // going outside the parent.
      const clampScale = scale.XY.clamp(
        box.construct(box.topLeft(TOP_LEFT_DECIMAL), {
          width: 1 - box.width(elDecimal),
          height: 1 - box.height(elDecimal),
        }),
      );
      const newDecimalPos = xy.translate(
        xy.construct(positionRef.current),
        box.signedDims(decimalScale.box(drag)),
      );
      const clamped = clampScale.pos(newDecimalPos);
      return calcStickyPos(clamped, ref);
    }, []);

    const handleCursorDragStart = useCursorDrag({
      onStart: useCallback(() => {
        // When we start dragging, we need to re-calculate the sticky position of the
        // element based on the new dimensions of the parent. This removes strange
        // 'jumping' behavior when starting to drag.
        positionRef.current = stickyToDecimalXY(positionRef.current, ref);
      }, []),
      onMove: useCallback((box: box.Box) => {
        if (disabled.current) return;
        const pos = calculatePosition(box);
        if (pos !== null) setPosition(pos);
      }, []),
      onEnd: useCallback((box: box.Box) => {
        if (disabled.current) return;
        const pos = calculatePosition(box);
        if (pos !== null) positionRef.current = pos;
      }, []),
    });

    return (
      <Align.Space
        className={CSS(className, CSS.B("legend"))}
        bordered
        rounded
        style={style}
        onDragStart={handleCursorDragStart}
        draggable={draggable}
        ref={ref}
        {...props}
        onDrag={preventDefault}
        onDragEnd={preventDefault}
        empty
      />
    );
  },
);
Container.displayName = "Legend.Container";
