// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/rule/Rule.css";

import { bounds, box } from "@synnaxlabs/x/spatial";
import { type ReactElement, useCallback, useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { CSS } from "@/css";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { state } from "@/state";
import { Text } from "@/text";
import { selectViewportEl } from "@/vis/lineplot/Viewport";
import { rule } from "@/vis/rule/aether";

export interface RuleProps
  extends Omit<z.input<typeof rule.ruleStateZ>, "dragging" | "pixelPosition"> {
  label?: string;
  onLabelChange?: (label: string) => void;
  units?: string;
  onPositionChange?: (position: number) => void;
}

export const Rule = Aether.wrap<RuleProps>(
  "Rule",
  ({
    aetherKey,
    label,
    position: propsPosition,
    onLabelChange,
    onPositionChange,
    units = "",
    color,
    lineWidth,
    lineDash,
  }): ReactElement | null => {
    const [internalLabel, setInternalLabel] = state.usePurePassthrough({
      value: label,
      onChange: onLabelChange,
      initial: "",
    });

    const [, { position, pixelPosition }, setState] = Aether.use({
      aetherKey,
      type: rule.Rule.TYPE,
      schema: rule.ruleStateZ,
      initialState: {
        color,
        dragging: false,
        position: propsPosition,
        lineWidth,
        lineDash,
      },
    });

    useEffect(() => {
      if (position == null) return;
      if (propsPosition == null) return onPositionChange?.(position);
      const b = bounds.construct(position + 0.01, position - 0.01);
      if (propsPosition != null && !bounds.contains(b, propsPosition))
        onPositionChange?.(Math.trunc(position * 100) / 100);
    }, [position]);

    const pixelPosRef = useRef(pixelPosition);
    if (pixelPosition !== pixelPosRef.current) pixelPosRef.current = pixelPosition;

    const dragStartRef = useRef(pixelPosition);

    useEffect(() => {
      setState((p) => ({ ...p, position: propsPosition, color, lineWidth, lineDash }));
    }, [propsPosition, color, lineWidth, lineDash]);

    const handleDragStart = useCursorDrag({
      onStart: useCallback(() => {
        setState((p) => ({ ...p, dragging: true }));
        dragStartRef.current = pixelPosRef.current;
      }, []),
      onMove: (b: box.Box) => {
        setState((p) => ({
          ...p,
          pixelPosition: (dragStartRef.current ?? 0) + box.signedHeight(b),
        }));
      },
      onEnd: useCallback(() => {
        setState((p) => ({ ...p, dragging: false }));
        dragStartRef.current = pixelPosition;
      }, []),
    });

    const ref = useRef<HTMLDivElement>(null);

    if (position == null || pixelPosition == null) return null;

    const viewportEl = selectViewportEl(ref?.current);

    return null;

    const content = (
      <div
        ref={ref}
        className={CSS.B("rule")}
        style={{ top: `calc(${pixelPosition}px - 0.5rem)` }}
      >
        <div
          className={CSS.BE("rule", "drag-handle")}
          onDragStart={handleDragStart}
          draggable
        />
        <Align.Space direction="x" align="center" style={{ marginLeft: "2rem" }}>
          <Text.Editable level="p" value={internalLabel} onChange={setInternalLabel} />
          <Text.Text
            level="p"
            style={{ padding: "0.25rem 0", width: "fit-content" }}
          >{`${position.toFixed(2)} ${units}`}</Text.Text>
        </Align.Space>
      </div>
    );

    if (viewportEl == null) return content;
    return createPortal(content, viewportEl);
  },
);
