// copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { useCallback, useRef, useState } from "react";

import {
  comparePrimitiveArrays,
  Box,
  Dimensions,
  DECIMAL_BOX,
  XY,
  ZERO_BOX,
  BoxScale,
} from "@synnaxlabs/x";

import { useMemoCompare } from "@/hooks";
import { useStateRef } from "@/hooks/useStateRef";
import { Stage, Trigger, TriggerDragCallback, Triggers } from "@/triggers";

export interface UseViewportEvent {
  box: Box;
  cursor: XY;
  mode: Mode;
  stage: Stage;
}

export type UseViewportHandler = (e: UseViewportEvent) => void;

export interface UseViewportTriggers {
  zoom?: Trigger[];
  zoomReset?: Trigger[];
  pan?: Trigger[];
  select?: Trigger[];
}

export interface UseViewportProps {
  defaultMode?: Mode;
  triggers?: UseViewportTriggers;
  onChange?: UseViewportHandler;
  resetOnDoubleClick?: boolean;
  threshold?: Dimensions;
}

export interface UseViewportReturn {
  mode: Mode;
  maskBox: Box;
  ref: React.RefObject<HTMLDivElement>;
}

export const MODES = ["zoom", "pan", "select", "zoomReset", null] as const;
type Mode = typeof MODES[number];
export const MASK_MODES: Mode[] = ["zoom", "select"];

const DEFAULT_TRIGGER_CONFIG: UseViewportTriggers = {
  zoom: [["MouseLeft", null]],
  zoomReset: [["MouseDouble", null]],
  pan: [["MouseLeft", "Shift"]],
  select: [["MouseLeft", "Alt"]],
};

const compareTriggerConfigs = (
  [a]: [UseViewportTriggers | undefined],
  [b]: [UseViewportTriggers | undefined]
): boolean => {
  if (a == null && b == null) return true;
  if (a == null || b == null) return false;
  return Object.entries(a).every(([key, value]) =>
    comparePrimitiveArrays(value, b[key as keyof UseViewportTriggers] as Trigger[])
  );
};

export const useViewport = ({
  onChange,
  defaultMode = "zoom",
  triggers: initialTriggers,
  threshold = { width: 30, height: 30 },
}: UseViewportProps): UseViewportReturn => {
  const [maskBox, setMaskBox] = useState<Box>(ZERO_BOX);
  const [stateRef, setStateRef] = useStateRef<Box>(DECIMAL_BOX);
  const canvasRef = useRef<HTMLDivElement>(null);

  const triggerConfig = useMemoCompare(
    () => ({
      ...initialTriggers,
      ...DEFAULT_TRIGGER_CONFIG,
    }),
    compareTriggerConfigs,
    [initialTriggers]
  );

  const handleDrag = useCallback<TriggerDragCallback>(
    ({ box, triggers, stage, cursor }): void => {
      if (canvasRef.current == null) return;
      const mode = determineMode(triggerConfig, triggers, defaultMode);
      const canvas = new Box(canvasRef.current);

      if (mode === "zoomReset") {
        setMaskBox(ZERO_BOX);
        onChange?.({ box: DECIMAL_BOX, mode, stage, cursor });
        return setStateRef(DECIMAL_BOX);
      }

      if (stage === "end") {
        if (box.width < 5 && box.height < 5) return;
        return setStateRef((prev) => {
          if (mode === "pan") {
            const next = handlePan(box, prev, canvas);
            if (next === null) return prev;
            onChange?.({ box: next, mode, stage, cursor });
            return next;
          }
          const next = handleZoomSelect(box, prev, canvas);
          if (next === null) return prev;
          onChange?.({ box: next, mode, stage, cursor });

          if (mode === "zoom") {
            setMaskBox(ZERO_BOX);
            return next;
          }
          return prev;
        });
      }

      if (MASK_MODES.includes(mode)) {
        return setMaskBox(
          BoxScale.scale(canvas)
            .clamp(canvas)
            .translate({
              x: -canvas.left,
              y: -canvas.top,
            })
            .box(fullSize(threshold, box, canvas))
        );
      }
      setMaskBox((prev) => (!prev.isZero ? ZERO_BOX : prev));
      onChange?.({
        box: handlePan(box, stateRef.current, canvas),
        mode,
        stage,
        cursor,
      });
    },
    [setMaskBox]
  );

  const handleZoomSelect = useCallback(
    (box: Box, prev: Box, canvas: Box): Box | null => {
      return scale(prev, canvas).box(fullSize(threshold, box, canvas));
    },
    [threshold]
  );

  Triggers.useDrag({
    bound: canvasRef,
    onDrag: handleDrag,
    triggers: reduceTriggerConfig(triggerConfig),
  });

  return {
    maskBox,
    ref: canvasRef,
    mode: "zoom",
  };
};

const scale = (prev: Box, canvas: Box): BoxScale =>
  BoxScale.scale(canvas).clamp(canvas).scale(prev);

const handlePan = (box: Box, prev: Box, canvas: Box): Box => {
  let dims = scale(prev, canvas).box(box).signedDims;
  dims = { signedWidth: -dims.signedWidth, signedHeight: -dims.signedHeight };
  return BoxScale.translate(dims).box(prev);
};

const fullSize = (threshold: Dimensions, box: Box, parent: Box): Box => {
  if (box.height <= threshold.height)
    return new Box(box.left, parent.top, box.width, parent.height);
  if (box.width <= threshold.width)
    return new Box(parent.left, box.top, parent.width, box.height);
  return box;
};

const determineMode = (
  config: UseViewportTriggers,
  triggers: Trigger[],
  defaultMode: Mode
): Mode => {
  if (config.zoom != null && Triggers.match(config.zoom, triggers)) return "zoom";
  if (config.pan != null && Triggers.match(config.pan, triggers)) return "pan";
  if (config.select != null && Triggers.match(config.select, triggers)) return "select";
  if (config.zoomReset != null && Triggers.match(config.zoomReset, triggers))
    return "zoomReset";
  return defaultMode;
};

const reduceTriggerConfig = (config: UseViewportTriggers): Trigger[] =>
  Object.values(config).flat();
