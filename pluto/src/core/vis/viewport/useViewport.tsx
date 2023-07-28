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
  defaultMode?: ViewportMode;
  zoom?: Trigger[];
  zoomReset?: Trigger[];
  pan?: Trigger[];
  select?: Trigger[];
  hover?: Trigger[];
}

export interface UseViewportProps {
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

export const ZOOM_DEFAULT_TRIGGERS: UseViewportTriggers = {
  defaultMode: "zoom",
  zoom: [["MouseLeft"]],
  zoomReset: [["MouseLeft", "MouseLeft"]],
  pan: [["MouseLeft", "Shift"]],
  select: [["MouseLeft", "Alt"]],
  hover: [[]],
};

export const PAN_DEFAULT_TRIGGERS: UseViewportTriggers = {
  defaultMode: "pan",
  pan: [["MouseLeft"]],
  zoom: [["MouseLeft", "Shift"]],
  zoomReset: [["MouseLeft", "MouseLeft"]],
  select: [["MouseLeft", "Alt"]],
  hover: [["H"]],
};

export const SELECT_DEFAULT_TRIGGERS: UseViewportTriggers = {
  defaultMode: "select",
  select: [["MouseLeft"]],
  pan: [["MouseLeft", "Shift"]],
  zoom: [["MouseLeft", "Alt"]],
  zoomReset: [["MouseLeft", "MouseLeft"]],
  hover: [["H"]],
};

export const DEFAULT_TRIGGERS: Record<ViewportMode, UseViewportTriggers> = {
  zoom: ZOOM_DEFAULT_TRIGGERS,
  pan: PAN_DEFAULT_TRIGGERS,
  select: SELECT_DEFAULT_TRIGGERS,
  zoomReset: ZOOM_DEFAULT_TRIGGERS,
  hover: ZOOM_DEFAULT_TRIGGERS,
};

const compareTriggers = (
  [a]: [UseViewportTriggers | undefined],
  [b]: [UseViewportTriggers | undefined]
): boolean => {
  if (a == null && b == null) return true;
  if (a == null || b == null) return false;
  if (a.defaultMode !== b.defaultMode) return false;
  const v = Object.entries(a)
    .filter(([k]) => k !== "defaultMode")
    .every(([key, value]: [string, Trigger[]]) => {
      const old = b[key as keyof UseViewportTriggers] as Trigger[];
      if (value.length !== old.length) return false;
      return value.every(
        (value, i) => Compare.primitiveArrays(value, old[i]) === Compare.equal
      );
    });
  return v;
};

const purgeMouseTriggers = (triggers: UseViewportTriggers): UseViewportTriggers =>
  Object.fromEntries(
    Object.entries(triggers)
      .filter(([key]) => key !== "defaultMode")
      .map(([key, value]: [string, Trigger[]]) => [
        key,
        value
          .map((t) => t.filter((k) => k !== "MouseLeft"))
          .filter((t) => t.length > 0),
      ])
  );

export const useViewport = ({
  onChange,
  triggers: initialTriggers,
  initial = Box.DECIMAL,
  threshold: threshold_ = { width: 30, height: 30 },
}: UseViewportProps): UseViewportReturn => {
  const defaultMode = initialTriggers?.defaultMode ?? "zoom";

  const [maskBox, setMaskBox] = useState<Box>(Box.ZERO);
  const [maskMode, setMaskMode] = useState<ViewportMode>(defaultMode);
  const [stateRef, setStateRef] = useStateRef<Box>(initial);
  const canvasRef = useRef<HTMLDivElement | null>(null);
  const threshold = new Dimensions(threshold_);

  useEffect(() => setStateRef(initial), [initial]);
  useEffect(() => setMaskMode(defaultMode), [defaultMode]);

  const [triggerConfig, reducedTriggerConfig, purgedTriggers, reducedPurgedTriggers] =
    useMemoCompare(
      (): [UseViewportTriggers, Trigger[], UseViewportTriggers, Trigger[]] => {
        const config: UseViewportTriggers = {
          ...DEFAULT_TRIGGERS[defaultMode],
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
      compareTriggers,
      [initialTriggers]
    );

  useEffect(() => {
    if (
      (triggerConfig.hover ?? []).every((t) => t.length > 0) ||
      canvasRef.current == null
    )
      return;
    canvasRef.current.addEventListener("mouseenter", () => {
      onChange?.({
        box: stateRef.current,
        mode: "hover",
        stage: "start",
        cursor: XY.ZERO,
      });
      setMaskMode("hover");
    });
    canvasRef.current.addEventListener("mouseleave", () => {
      if (maskMode === "hover")
        onChange?.({
          box: stateRef.current,
          mode: "hover",
          stage: "end",
          cursor: XY.ZERO,
        });
      setMaskMode(defaultMode);
    });
    canvasRef.current.addEventListener("mousemove", (e) => {
      onChange?.({
        box: stateRef.current,
        mode: "hover",
        stage: "during",
        cursor: new XY({ x: e.clientX, y: e.clientY }),
      });
    });
  }, [triggerConfig]);

  const handleDrag = useCallback<TriggerDragCallback>(
    ({ box, triggers, stage, cursor }): void => {
      if (canvasRef.current == null) return;
      const mode = determineMode(triggerConfig, triggers, defaultMode);
      const canvas = new Box(canvasRef.current);
      if (mode == null) return;

      if (mode === "hover")
        return onChange?.({ box: stateRef.current, mode, stage, cursor });

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

      if (MASK_VIEWPORT_MODES.includes(mode)) {
        if (box.height < 5 && box.width < 5) return;
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
    [purgedTriggers, defaultMode, onChange]
  );

  Triggers.use({
    triggers: reducedPurgedTriggers,
    callback: handleKeyTrigger,
    loose: true,
    region: canvasRef,
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
  if (config.zoomReset != null && Triggers.match(config.zoomReset, triggers))
    return "zoomReset";
  if (config.zoom != null && Triggers.match(config.zoom, triggers)) return "zoom";
  if (config.hover != null && Triggers.match(config.hover, triggers)) return "hover";
  return defaultMode;
};

const reduceTriggerConfig = (config: UseViewportTriggers): Trigger[] =>
  Object.values(config).flat();
