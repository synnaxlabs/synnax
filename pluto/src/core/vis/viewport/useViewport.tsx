// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useRef, useState } from "react";

import { Box, Dimensions, XY, XYScale, CrudeDimensions } from "@synnaxlabs/x";

import { useMemoCompare } from "@/core/hooks";
import { useStateRef } from "@/core/hooks/useStateRef";
import {
  Stage,
  Trigger,
  TriggerDragCallback,
  Triggers,
  UseTriggerEvent,
} from "@/core/triggers";
import {
  TriggerConfig,
  compareTriggerConfigs,
  determineTriggerMode,
  reduceTriggerConfig,
} from "@/core/triggers/triggers";

export interface UseViewportEvent {
  box: Box;
  cursor: XY;
  mode: ViewportMode;
  stage: Stage;
}

export type UseViewportHandler = (e: UseViewportEvent) => void;

export type UseViewportTriggers = TriggerConfig<ViewportTriggerMode>;

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

type StringLiteral<T> = T extends string ? (string extends T ? never : T) : never;

const VIEWPORT_TRIGGER_MDOES = ["zoom", "pan", "select", "zoomReset"] as const;
export const VIEWPORT_MODES = [...VIEWPORT_TRIGGER_MDOES, "click"] as const;
export type ViewportMode = StringLiteral<typeof VIEWPORT_MODES[number]>;
type ViewportTriggerMode = StringLiteral<typeof VIEWPORT_TRIGGER_MDOES[number]>;
export const MASK_VIEWPORT_MODES: ViewportMode[] = ["zoom", "select"];

export const ZOOM_DEFAULT_TRIGGERS: UseViewportTriggers = {
  defaultMode: "zoom",
  zoom: [["MouseLeft"]],
  zoomReset: [["MouseLeft", "Meta"]],
  pan: [["MouseLeft", "Shift"]],
  select: [["MouseLeft", "Alt"]],
};

export const PAN_DEFAULT_TRIGGERS: UseViewportTriggers = {
  defaultMode: "pan",
  pan: [["MouseLeft"]],
  zoom: [["MouseLeft", "Shift"]],
  zoomReset: [["MouseLeft", "Meta"]],
  select: [["MouseLeft", "Alt"]],
};

export const SELECT_DEFAULT_TRIGGERS: UseViewportTriggers = {
  defaultMode: "select",
  select: [["MouseLeft"]],
  pan: [["MouseLeft", "Shift"]],
  zoom: [["MouseLeft", "Alt"]],
  zoomReset: [["MouseLeft", "Meta"]],
};

export const DEFAULT_TRIGGERS: Record<ViewportMode, UseViewportTriggers> = {
  zoom: ZOOM_DEFAULT_TRIGGERS,
  pan: PAN_DEFAULT_TRIGGERS,
  select: SELECT_DEFAULT_TRIGGERS,
  zoomReset: ZOOM_DEFAULT_TRIGGERS,
  click: ZOOM_DEFAULT_TRIGGERS,
};

const purgeMouseTriggers = (triggers: UseViewportTriggers): UseViewportTriggers => {
  const e = Object.entries(triggers) as Array<
    [ViewportTriggerMode | "defaultMode", Trigger[]]
  >;
  return Object.fromEntries(
    e.map(([key, value]: [string, Trigger[]]) => {
      if (key === "defaultMode") return [key, value];
      return [
        key,
        value
          .map((t) => t.filter((k) => k !== "MouseLeft"))
          .filter((t) => t.length > 0),
      ];
    })
  ) as unknown as UseViewportTriggers;
};

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
      compareTriggerConfigs,
      [initialTriggers]
    );

  const handleDrag = useCallback<TriggerDragCallback>(
    ({ box, triggers, stage, cursor }): void => {
      if (canvasRef.current == null) return;
      const mode = determineTriggerMode<ViewportTriggerMode>(triggerConfig, triggers);
      const canvas = new Box(canvasRef.current);
      if (mode == null) return;

      if (mode === "zoomReset") {
        setMaskBox(Box.ZERO);
        onChange?.({ box: Box.DECIMAL, mode, stage, cursor });
        return setStateRef(Box.DECIMAL);
      }

      if (stage === "end") {
        // This prevents clicks from being registered as a drag
        if (box.width < 5 && box.height < 5) {
          if (mode === "zoom") setMaskBox(Box.ZERO);
          onChange?.({ box: stateRef.current, mode: "click", stage, cursor });
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
          XYScale.scale(canvas)
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
    loose: true,
  });

  const handleKeyTrigger = useCallback(
    ({ triggers, stage }: UseTriggerEvent) => {
      if (stage === "end") return setMaskMode(defaultMode);
      const mode = determineTriggerMode<ViewportTriggerMode>(purgedTriggers, triggers);
      if (mode == null) return;
      setMaskMode(mode);
    },
    [purgedTriggers, defaultMode, onChange]
  );

  Triggers.use({
    triggers: reducedPurgedTriggers,
    callback: handleKeyTrigger,
    // loose: true,
    region: canvasRef,
  });

  return {
    maskBox,
    ref: canvasRef,
    mode: maskMode,
  };
};

const scale = (prev: Box, canvas: Box): XYScale =>
  XYScale.scale(canvas).clamp(canvas).scale(prev);

const handlePan = (box: Box, prev: Box, canvas: Box): Box => {
  let dims = scale(prev, canvas).box(box).signedDims;
  dims = { signedWidth: -dims.signedWidth, signedHeight: -dims.signedHeight };
  return XYScale.translate(dims).box(prev);
};

const fullSize = (threshold: Dimensions, box: Box, parent: Box): Box => {
  if (box.height <= threshold.height)
    return new Box(box.left, parent.top, box.width, parent.height);
  if (box.width <= threshold.width)
    return new Box(parent.left, box.top, parent.width, box.height);
  return box;
};
