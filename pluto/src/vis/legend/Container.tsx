// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type CSSProperties,
  memo,
  type ReactElement,
  type RefObject,
  useCallback,
  useRef,
  useState,
} from "react";

import { box, location, scale, xy } from "@synnaxlabs/x";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { type OptionalControl } from "@/input/types";
import { state } from "@/state";
import { preventDefault } from "@/util/event";

import "@/vis/legend/Container.css";

export interface ContainerProps
  extends Omit<Align.SpaceProps, "onChange">,
    Partial<OptionalControl<xy.XY>> {
  dragEnabled?: boolean;
}

type CSSPosition = Partial<
  Pick<CSSProperties, "left" | "right" | "top" | "bottom" | "display">
>;

export const intelligentPosition = (
  pos: xy.XY,
  ref: RefObject<HTMLDivElement>,
): CSSPosition => {
  if (ref.current == null) return { display: "none" };
  const ret: CSSPosition = {};
  const parentBox = box.construct(ref.current.parentElement as HTMLDivElement);
  const b = box.construct(ref.current);
  if (pos.x > 0.8) {
    ret.right = `${(1 - pos.x) * box.width(parentBox) - box.width(b)}px`;
  } else if (pos.x < 0.2) {
    ret.left = `${pos.x * box.width(parentBox)}px`;
  } else {
    ret.left = `${pos.x * 100}%`;
  }
  if (pos.y > 0.8) {
    ret.bottom = `${(1 - pos.y) * box.height(parentBox) - box.height(b)}px`;
  } else if (pos.y < 0.2) {
    ret.top = `${pos.y * box.height(parentBox)}px`;
  } else {
    ret.top = `${pos.y * 100}%`;
  }
  return ret;
};

export const Container = memo(
  ({
    className,
    value,
    onChange,
    style,
    draggable = true,
    ...props
  }: ContainerProps): ReactElement | null => {
    const [position, setPosition] = state.usePurePassthrough<xy.XY>({
      value,
      onChange,
      initial: xy.construct(0.1, 0.01),
    });

    const positionRef = useRef(position);
    const disabled = useSyncedRef(!draggable);
    const ref = useRef<HTMLDivElement | null>(null);
    const [intelligentPos, setIntelligentPos] = useState<CSSPosition>({});

    if (position !== null) style = { ...style, ...intelligentPos };

    const refCallback = useCallback((el: HTMLDivElement | null) => {
      ref.current = el;
      setIntelligentPos(intelligentPosition(xy.construct(position), ref));
    }, []);

    const calculatePosition = useCallback((b: box.Box): xy.XY => {
      if (ref.current?.parentElement == null) return positionRef.current;
      const bounds = box.construct(ref.current.parentElement);
      const d = box.reRoot(box.DECIMAL, location.TOP_LEFT);
      const s = scale.XY.scale(bounds).scale(d);
      const el = s.box(box.construct(ref.current));
      const clamp = scale.XY.clamp(
        box.construct(box.topLeft(d), {
          width: box.width(d) - box.width(el),
          height: box.height(d) - box.height(el),
        }),
      );
      return clamp.pos(
        xy.translate(xy.construct(positionRef.current), box.signedDims(s.box(b))),
      );
    }, []);

    const handleCursorDragStart = useCursorDrag({
      onMove: useCallback(
        (box: box.Box) => {
          if (disabled.current) return;
          const pos = calculatePosition(box);
          setIntelligentPos(intelligentPosition(xy.construct(pos), ref));
        },
        [setPosition],
      ),
      onEnd: useCallback((box: box.Box) => {
        if (disabled.current) return;
        positionRef.current = calculatePosition(box);
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
        ref={refCallback}
        {...props}
        onDrag={preventDefault}
        onDragEnd={preventDefault}
        size="small"
      />
    );
  },
);
Container.displayName = "Legend.Container";
