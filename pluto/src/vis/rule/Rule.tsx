// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/rule/Rule.css";

import { box, color } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { LinePlot } from "@/lineplot";
import { state } from "@/state";
import { Text } from "@/text";
import { rule } from "@/vis/rule/aether";

export interface RuleProps
  extends
    Omit<z.input<typeof rule.ruleStateZ>, "dragging" | "pixelPosition">,
    Omit<Flex.BoxProps, "color">,
    Aether.ComponentProps {
  label?: string;
  onLabelChange?: (label: string) => void;
  units?: string;
  onUnitsChange?: (label: string) => void;
  onPositionChange?: (position: number) => void;
  onSelect?: () => void;
}

export const Rule = ({
  aetherKey,
  label,
  position: propsPosition,
  onLabelChange,
  onPositionChange,
  onUnitsChange,
  units = "",
  color: colorVal,
  lineWidth,
  lineDash,
  className,
  onSelect,
  style,
  ...rest
}: RuleProps): ReactElement | null => {
  const [internalLabel, setInternalLabel] = state.usePurePassthrough({
    value: label,
    onChange: onLabelChange,
    initial: "",
  });

  const onPositionChangeRef = useSyncedRef(onPositionChange);

  const [, { pixelPosition }, setState] = Aether.use({
    aetherKey,
    type: rule.Rule.TYPE,
    schema: rule.ruleStateZ,
    initialState: {
      color: colorVal,
      dragging: false,
      position: propsPosition,
      lineWidth,
      lineDash,
    },
    onAetherChange: useCallback(
      ({ position }: z.infer<typeof rule.ruleStateZ>) => {
        if (position == null) return;
        onPositionChangeRef.current?.(position);
      },
      [onPositionChangeRef],
    ),
  });

  const pixelPosRef = useRef(pixelPosition);
  if (pixelPosition !== pixelPosRef.current) pixelPosRef.current = pixelPosition;

  const { id } = LinePlot.useContext("Rule.Rule");

  const plotEl = document.getElementById(id);
  const viewportEl = plotEl?.querySelector(".pluto-line-plot__viewport");

  const dragStartRef = useRef(pixelPosition);

  useEffect(() => {
    setState((p) => ({
      ...p,
      position: propsPosition,
      color: colorVal,
      lineWidth,
      lineDash,
    }));
  }, [propsPosition, colorVal, lineWidth, lineDash]);

  const handleDragStart = useCursorDrag({
    onStart: useCallback(() => {
      onSelect?.();
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

  if (propsPosition == null || pixelPosition == null) return null;

  const textColor = color.pickByContrast(colorVal, color.BLACK, color.WHITE);

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
      <Flex.Box
        x
        align="center"
        className={CSS(className, CSS.BE("rule", "tag"))}
        bordered
        onClick={onSelect}
        empty
        rounded
        style={{
          borderColor: color.cssString(colorVal),
          backgroundColor: color.hex(color.setAlpha(colorVal, 0.7)),
          ...style,
        }}
        {...rest}
      >
        <Text.Editable
          className={CSS.BE("rule", "label")}
          level="small"
          value={internalLabel}
          onChange={setInternalLabel}
          color={textColor}
        />
        <Divider.Divider y style={{ borderColor: color.cssString(colorVal) }} />
        <Flex.Box x align="center" className={CSS.BE("rule", "value")}>
          <Text.Editable
            value={propsPosition.toFixed(2)}
            onChange={(v) => {
              const num = Number(v);
              if (!isFinite(num)) return;
              onPositionChange?.(num);
            }}
            level="small"
            color={textColor}
          />
          {units.length > 0 && (
            <Text.MaybeEditable
              level="small"
              color={textColor}
              value={units}
              onChange={onUnitsChange}
            />
          )}
        </Flex.Box>
      </Flex.Box>
    </div>
  );

  if (viewportEl == null) return content;
  return createPortal(content, viewportEl);
};
