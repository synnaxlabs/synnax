// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useRef, useState } from "react";

import { Box, Dimensions, XY, BoxScale, Compare, CrudeDimensions } from "@synnaxlabs/x";

import { useMemoCompare } from "@/core/hooks";
import { useStateRef } from "@/core/hooks/useStateRef";
import {
  Stage,
  Trigger,
  TriggerCallback,
  TriggerDragCallback,
  Triggers,
  UseTriggerEvent,
} from "@/core/triggers";

export interface UseViewportEvent {
  box: Box;
  cursor: XY;
  mode: ViewportMode;
  stage: Stage;
}

export type UseViewportHandler = (e: UseViewportEvent) => void;

export interface UseViewportTriggers {
  zoom?: Trigger[];
  zoomReset?: Trigger[];
  pan?: Trigger[];
  select?: Trigger[];
  hover?: Trigger[];
}

export interface UseViewportProps {
  defaultMode?: ViewportMode;
  triggers?: UseViewportTriggers;
  onChange?: UseViewportHandler;
  resetOnDoubleClick?: boolean;
  threshold?: CrudeDimensions;
  initial?: Box;
}

export interface UseViewportReturn {
  mode: ViewportMode;
  maskBox: Box;
  ref: React.MutableRefObject<HTMLDivElement | null>;
}

export const VIEWPORT_MODES = ["zoom", "pan", "select", "zoomReset", "hover"] as const;
export type ViewportMode = typeof VIEWPORT_MODES[number];
export const MASK_VIEWPORT_MODES: ViewportMode[] = ["zoom", "select"];

const DEFAULT_TRIGGER_CONFIG: UseViewportTriggers = {
  zoom: [["MouseLeft"]],
  zoomReset: [["MouseLeft", "MouseLeft"]],
  pan: [["MouseLeft", "Shift"]],
  select: [["MouseLeft", "Alt"]],
  hover: [["T"]],
};

const compareTriggerConfigs = (
  [a]: [UseViewportTriggers | undefined],
  [b]: [UseViewportTriggers | undefined]
): boolean => {
  if (a == null && b == null) return true;
  if (a == null || b == null) return false;
  return Object.entries(a).every(([key, value]) =>
    Compare.primitiveArrays(value, b[key as keyof UseViewportTriggers] as Trigger[])
  );
};

const purgeMouseTriggers = (triggers: UseViewportTriggers): UseViewportTriggers =>
  Object.fromEntries(
    Object.entries(triggers).map(([key, value]: [string, Trigger[]]) => [
      key,
      value.map((t) => t.filter((k) => k !== "MouseLeft")),
    ])
  );

export const useViewport = ({
  onChange,
  defaultMode = "zoom",
  triggers: initialTriggers,
  initial = Box.DECIMAL,
  threshold: threshold_ = { width: 30, height: 30 },
}: UseViewportProps): UseViewportReturn => {
  const [maskBox, setMaskBox] = useState<Box>(Box.ZERO);
  const [maskMode, setMaskMode] = useState<ViewportMode>(defaultMode);
  const [stateRef, setStateRef] = useStateRef<Box>(initial);
  const canvasRef = useRef<HTMLDivElement | null>(null);
  const threshold = new Dimensions(threshold_);

  useEffect(() => setStateRef(initial), [initial]);

  const [triggerConfig, reducedTriggerConfig, purgedTriggers, reducedPurgedTriggers] =
    useMemoCompare(
      (): [UseViewportTriggers, Trigger[], UseViewportTriggers, Trigger[]] => {
        const config: UseViewportTriggers = {
          ...DEFAULT_TRIGGER_CONFIG,
          ...initialTriggers,
        };
        const reducedTriggers = reduceTriggerConfig(config);
        const mouseTriggers = purgeMouseTriggers(config);
        return [
          config,
          reducedTriggers,
          mouseTriggers,
          reduceTriggerConfig(mouseTriggers),
        ];
      },
      compareTriggerConfigs,
      [initialTriggers]
    );

  const handleDrag = useCallback<TriggerDragCallback>(
    ({ box, triggers, stage, cursor }): void => {
      if (canvasRef.current == null) return;
      const mode = determineMode(triggerConfig, triggers, defaultMode);
      const canvas = new Box(canvasRef.current);
      if (mode == null) return;

      if (mode === "hover")
        return setStateRef((prev) => {
          // Point box at cursor
          const next = handleHover(prev, canvas, cursor);
          onChange?.({ box: next, mode, stage, cursor });
          return prev;
        });

      if (mode === "zoomReset") {
        setMaskBox(Box.ZERO);
        onChange?.({ box: Box.DECIMAL, mode, stage, cursor });
        return setStateRef(Box.DECIMAL);
      }

      if (stage === "end") {
        // This prevents clicks from being registered as a drag
        if (box.width < 5 && box.height < 5) {
          if (mode === "zoom") setMaskBox(Box.ZERO);
          return;
        }
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
            setMaskBox(Box.ZERO);
            return next;
          }
          return prev;
        });
      }

      if (MASK_VIEWPORT_MODES.includes(mode))
        return setMaskBox(
          BoxScale.scale(canvas)
            .clamp(canvas)
            .translate({
              x: -canvas.left,
              y: -canvas.top,
            })
            .box(fullSize(threshold, box, canvas))
        );

      setMaskBox((prev) => (!prev.isZero ? Box.ZERO : prev));
      onChange?.({
        box: handlePan(box, stateRef.current, canvas),
        mode,
        stage,
        cursor,
      });
    },
    [setMaskBox, setMaskMode, onChange, triggerConfig]
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
    triggers: reducedTriggerConfig,
  });

  const handleKeyTrigger = useCallback(
    ({ triggers, stage }: UseTriggerEvent) => {
      if (stage === "end") return setMaskMode(defaultMode);
      const mode = determineMode(purgedTriggers, triggers, defaultMode);
      if (mode == null) return;
      setMaskMode(mode);
    },
    [purgedTriggers, defaultMode]
  );

  Triggers.use({
    triggers: reducedPurgedTriggers,
    callback: handleKeyTrigger,
    loose: true,
  });

  return {
    maskBox,
    ref: canvasRef,
    mode: maskMode,
  };
};

const scale = (prev: Box, canvas: Box): BoxScale =>
  BoxScale.scale(canvas).clamp(canvas).scale(prev);

const handlePan = (box: Box, prev: Box, canvas: Box): Box => {
  let dims = scale(prev, canvas).box(box).signedDims;
  dims = { signedWidth: -dims.signedWidth, signedHeight: -dims.signedHeight };
  return BoxScale.translate(dims).box(prev);
};

const handleHover = (prev: Box, canvas: Box, cursor: XY): Box => {
  const dims = scale(prev, canvas);
  return dims.box(new Box(cursor, Dimensions.ZERO));
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
  defaultMode: ViewportMode
): ViewportMode => {
  if (config.pan != null && Triggers.match(config.pan, triggers)) return "pan";
  if (config.select != null && Triggers.match(config.select, triggers)) return "select";
  if (config.hover != null && Triggers.match(config.hover, triggers)) return "hover";
  if (config.zoomReset != null && Triggers.match(config.zoomReset, triggers))
    return "zoomReset";
  if (config.zoom != null && Triggers.match(config.zoom, triggers)) return "zoom";
  return defaultMode;
};

const reduceTriggerConfig = (config: UseViewportTriggers): Trigger[] =>
  Object.values(config).flat();
