// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/legend/Container.css";

import { box, location, scale, sticky, xy } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useMemo, useRef } from "react";

import { CSS } from "@/css";
import { Cursor } from "@/cursor";
import { Flex } from "@/flex";
import { useSyncedRef } from "@/hooks";
import { type OptionalControl } from "@/input/types";
import { state } from "@/state";

export interface ContainerProps
  extends Omit<Flex.BoxProps, "onChange">, Partial<OptionalControl<sticky.XY>> {
  dragEnabled?: boolean;
  initial?: sticky.XY;
}

const TOP_LEFT_DECIMAL = box.reRoot(box.DECIMAL, location.TOP_LEFT);

const DEFAULT_INITIAL: sticky.XY = {
  x: 0.1,
  y: 0.1,
  root: location.TOP_LEFT,
  units: { x: "decimal", y: "decimal" },
};

export const Container = memo(
  ({
    className,
    value,
    onChange,
    style,
    draggable = true,
    initial = DEFAULT_INITIAL,
    ...rest
  }: ContainerProps): ReactElement | null => {
    const [position, setPosition] = state.usePurePassthrough<sticky.XY>({
      value,
      onChange,
      initialValue: initial,
    });
    const positionRef = useRef<sticky.XY>(position);
    const disabled = useSyncedRef(draggable === false);
    const ref = useRef<HTMLDivElement | null>(null);

    const mergedStyle = useMemo(
      () => (position !== null ? { ...style, ...sticky.toCSS(position) } : style),
      [position, style],
    );

    const calculatePosition = useCallback((drag: box.Box): sticky.XY | null => {
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
      if (ref.current == null || ref.current.parentElement == null)
        return positionRef.current;
      return sticky.calculate({
        position: clamped,
        element: box.construct(ref.current),
        container: bounds,
      });
    }, []);

    const handleCursorDragStart = Cursor.useDrag({
      onStart: useCallback(() => {
        // When we start dragging, we need to re-calculate the sticky position of the
        // element based on the new dimensions of the parent. This removes strange
        // 'jumping' behavior when starting to drag.
        if (ref.current == null || ref.current.parentElement == null) return;
        positionRef.current = {
          ...sticky.toDecimal({
            position: positionRef.current,
            element: box.construct(ref.current),
            container: box.construct(ref.current.parentElement),
          }),
          root: location.TOP_LEFT,
          units: { x: "decimal", y: "decimal" },
        };
      }, []),
      onMove: useCallback(
        (box: box.Box) => {
          if (disabled.current) return;
          const pos = calculatePosition(box);
          if (pos !== null) setPosition(pos);
        },
        [calculatePosition, setPosition],
      ),
      onEnd: useCallback(
        (box: box.Box) => {
          if (disabled.current) return;
          const pos = calculatePosition(box);
          if (pos !== null) positionRef.current = pos;
        },
        [calculatePosition],
      ),
    });

    return (
      <Flex.Box
        className={CSS(className, CSS.B("legend"), Cursor.DRAG_CLASS)}
        bordered
        style={mergedStyle}
        onPointerDown={draggable ? handleCursorDragStart : undefined}
        borderColor={5}
        ref={ref}
        background={1}
        rounded={1}
        {...rest}
      />
    );
  },
);
Container.displayName = "Legend.Container";
