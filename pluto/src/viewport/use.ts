// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useRef, useState } from "react";

import {
  Box,
  Dimensions,
  XY,
  XYScale,
  CrudeDimensions,
  XYLocation,
} from "@synnaxlabs/x";

import { useMemoCompare } from "@/hooks";
import { useStateRef } from "@/hooks/useStateRef";
import { Triggers } from "@/triggers";
import {
  Config,
  compareConfigs,
  determineMode,
  reduceConfig,
} from "@/triggers/triggers";

export interface UseEvent {
  box: Box;
  cursor: XY;
  mode: Mode;
  stage: Triggers.Stage;
}

export type UseHandler = (e: UseEvent) => void;

export type UseTriggers = Config<TriggerMode>;

export interface UseProps {
  triggers?: UseTriggers;
  onChange?: UseHandler;
  resetOnDoubleClick?: boolean;
  threshold?: CrudeDimensions;
  initial?: Box;
}

export interface UseReturn {
  mode: Mode;
  maskBox: Box;
  ref: React.MutableRefObject<HTMLDivElement | null>;
}

type StringLiteral<T> = T extends string ? (string extends T ? never : T) : never;

const TRIGGER_MODES = ["zoom", "pan", "select", "zoomReset"] as const;
export const MODES = [...TRIGGER_MODES, "click"] as const;
export type Mode = StringLiteral<typeof MODES[number]>;
type TriggerMode = StringLiteral<typeof TRIGGER_MODES[number]>;
export const MASK_MODES: Mode[] = ["zoom", "select"];

export const ZOOM_DEFAULT_TRIGGERS: UseTriggers = {
  defaultMode: "zoom",
  zoom: [["MouseLeft"]],
  zoomReset: [["MouseLeft", "Control"]],
  pan: [["MouseLeft", "Shift"]],
  select: [["MouseLeft", "Alt"]],
};

export const PAN_DEFAULT_TRIGGERS: UseTriggers = {
  defaultMode: "pan",
  pan: [["MouseLeft"]],
  zoom: [["MouseLeft", "Shift"]],
  zoomReset: [["MouseLeft", "Control"]],
  select: [["MouseLeft", "Alt"]],
};

export const SELECT_DEFAULT_TRIGGERS: UseTriggers = {
  defaultMode: "select",
  select: [["MouseLeft"]],
  pan: [["MouseLeft", "Shift"]],
  zoom: [["MouseLeft", "Alt"]],
  zoomReset: [["MouseLeft", "Control"]],
};

export const DEFAULT_TRIGGERS: Record<Mode, UseTriggers> = {
  zoom: ZOOM_DEFAULT_TRIGGERS,
  pan: PAN_DEFAULT_TRIGGERS,
  select: SELECT_DEFAULT_TRIGGERS,
  zoomReset: ZOOM_DEFAULT_TRIGGERS,
  click: ZOOM_DEFAULT_TRIGGERS,
};

const purgeMouseTriggers = (triggers: UseTriggers): UseTriggers => {
  const e = Object.entries(triggers) as Array<
    [TriggerMode | "defaultMode", Triggers.Trigger[]]
  >;
  return Object.fromEntries(
    e.map(([key, value]: [string, Triggers.Trigger[]]) => {
      if (key === "defaultMode") return [key, value];
      return [
        key,
        value
          .map((t) => t.filter((k) => k !== "MouseLeft"))
          .filter((t) => t.length > 0),
      ];
    })
  ) as unknown as UseTriggers;
};

const D = new Box(0, 0, 1, 1, XYLocation.TOP_LEFT);

const DEFAULT_THRESHOLD = { width: 30, height: 30 };

export const use = ({
  onChange,
  triggers: initialTriggers,
  initial = D,
  threshold: threshold_ = DEFAULT_THRESHOLD,
}: UseProps): UseReturn => {
  const defaultMode = initialTriggers?.defaultMode ?? "zoom";

  const [maskBox, setMaskBox] = useState<Box>(Box.ZERO);
  const [maskMode, setMaskMode] = useState<Mode>(defaultMode);
  const [stateRef, setStateRef] = useStateRef<Box>(initial);
  const canvasRef = useRef<HTMLDivElement | null>(null);
  const threshold = new Dimensions(threshold_);

  useEffect(() => setStateRef(initial), [initial]);
  useEffect(() => setMaskMode(defaultMode), [defaultMode]);

  const [triggerConfig, reducedTriggerConfig, purgedTriggers, reducedPurgedTriggers] =
    useMemoCompare(
      (): [UseTriggers, Triggers.Trigger[], UseTriggers, Triggers.Trigger[]] => {
        const config: UseTriggers = {
          ...DEFAULT_TRIGGERS[defaultMode],
          ...initialTriggers,
        };
        const reducedTriggers = reduceConfig(config);
        const mouseTriggers = purgeMouseTriggers(config);
        return [config, reducedTriggers, mouseTriggers, reduceConfig(mouseTriggers)];
      },
      Triggers.compareConfigs,
      [initialTriggers]
    );

  const handleDrag = useCallback<Triggers.DragCallback>(
    ({ box, triggers, stage, cursor }): void => {
      if (canvasRef.current == null) return;
      const mode = determineMode<TriggerMode>(triggerConfig, triggers);
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

      if (MASK_MODES.includes(mode)) {
        if (box.height < 5 && box.width < 5) return;
        return setMaskBox(
          XYScale.scale(canvas)
            .clamp(canvas)
            .translate({ x: -canvas.left, y: -canvas.top })
            .box(fullSize(threshold, box, canvas))
        );
      }

      setMaskBox((prev) => (!prev.isZero ? Box.ZERO : prev));
      const next = handlePan(box, stateRef.current, canvas);
      onChange?.({
        box: next,
        mode,
        stage,
        cursor,
      });
    },
    [
      setMaskBox,
      setMaskMode,
      onChange,
      triggerConfig,
      threshold_.height,
      threshold_.width,
      setStateRef,
      canvasRef,
    ]
  );

  const handleZoomSelect = useCallback(
    (box: Box, prev: Box, canvas: Box): Box | null => {
      return scale(prev, canvas).box(fullSize(threshold, box, canvas));
    },
    [threshold_]
  );

  Triggers.useDrag({
    bound: canvasRef,
    onDrag: handleDrag,
    triggers: reducedTriggerConfig,
    loose: true,
  });

  const handleKeyTrigger = useCallback(
    ({ triggers, stage }: Triggers.UseEvent) => {
      if (stage === "end") return setMaskMode(defaultMode);
      const mode = determineMode<TriggerMode>(purgedTriggers, triggers);
      if (mode == null) return;
      setMaskMode(mode);
    },
    [purgedTriggers, defaultMode, onChange]
  );

  Triggers.use({
    triggers: reducedPurgedTriggers,
    callback: handleKeyTrigger,
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
