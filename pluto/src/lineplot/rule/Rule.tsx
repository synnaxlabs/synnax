// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/rule/Rule.css";

import { Cursor } from "@synnaxlabs/lyra/cursor";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { state } from "@synnaxlabs/lyra/state";
import { Text } from "@synnaxlabs/lyra/text";
import { box, color } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { useContext } from "@/lineplot/Frame";
import { rule } from "@/lineplot/rule/aether";

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
    initialValue: "",
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
    onAetherChange: ({ position }: z.infer<typeof rule.ruleStateZ>) => {
      if (position == null) return;
      onPositionChangeRef.current?.(position);
    },
  });

  const pixelPosRef = useSyncedRef(pixelPosition);

  const { viewportElement } = useContext("Rule.Rule");

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

  const handleDragStart = Cursor.useDrag({
    onStart: () => {
      onSelect?.();
      setState((p) => ({ ...p, dragging: true }));
      dragStartRef.current = pixelPosRef.current;
    },
    onMove: (b: box.Box) => {
      setState((p) => ({
        ...p,
        pixelPosition: (dragStartRef.current ?? 0) + box.signedHeight(b),
      }));
    },
    onEnd: () => {
      setState((p) => ({ ...p, dragging: false }));
    },
  });

  const ref = useRef<HTMLDivElement>(null);

  const topStyle = { top: `calc(${pixelPosition}px - 0.5rem)` };

  const borderColor = color.cssString(colorVal);
  const colorStyles = {
    tag: {
      borderColor,
      backgroundColor: color.hex(color.setAlpha(colorVal, 0.75)),
      ...style,
    },
    divider: { borderColor },
  };

  if (propsPosition == null || pixelPosition == null) return null;

  const textColor = color.pickByContrast(colorVal, color.BLACK, color.WHITE);

  const content = (
    <div ref={ref} className={CSS.B("rule")} style={topStyle}>
      <div
        className={CSS.cls(CSS.BE("rule", "drag-handle"), Cursor.DRAG_CLASS)}
        onPointerDown={handleDragStart}
      />
      <Flex.Box
        x
        align="center"
        className={CSS.cls(className, CSS.BE("rule", "tag"))}
        bordered
        onClick={onSelect}
        empty
        rounded
        style={colorStyles.tag}
        {...rest}
      >
        <Text.Editable
          className={CSS.BE("rule", "label")}
          level="small"
          value={internalLabel}
          onChange={setInternalLabel}
          color={textColor}
        />
        <Divider.Divider y style={colorStyles.divider} />
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

  if (viewportElement == null) return content;
  return createPortal(content, viewportElement);
};
