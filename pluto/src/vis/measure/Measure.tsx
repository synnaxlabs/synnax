// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useRef } from "react";

import { Aether } from "@/aether";
import { useSyncedRef } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { LinePlot } from "@/lineplot";
import { Triggers } from "@/triggers";
import { type Viewport } from "@/viewport";
import { measure } from "@/vis/measure/aether";

const MEASURE_TRIGGERS: Triggers.ModeConfig<measure.Mode> = {
  one: [["1"]],
  two: [["2"]],
  defaultMode: "one",
};

const REDUCED_MEASURE_TRIGGERS = Triggers.flattenConfig(MEASURE_TRIGGERS);

export interface MeasureProps extends Aether.ComponentProps {
  mode?: measure.Mode;
  onModeChange?: (mode: measure.Mode) => void;
}

const determineMode = (triggers: Triggers.Trigger[]): measure.Mode =>
  Triggers.determineMode<measure.Mode>(MEASURE_TRIGGERS, triggers, { loose: true });

export const Measure = ({
  aetherKey,
  mode = "one",
  onModeChange,
}: MeasureProps): ReactElement => {
  const cKey = useUniqueKey(aetherKey);
  const [, state, setState] = Aether.use({
    aetherKey: cKey,
    type: measure.Measure.TYPE,
    schema: measure.measureStateZ,
    initialState: { hover: null, one: null, two: null, mode },
  });

  useEffect(() => setState((p) => ({ ...p, mode })), [mode]);

  const ref = useRef<HTMLSpanElement>(null);

  Triggers.use({
    triggers: REDUCED_MEASURE_TRIGGERS,
    loose: true,
    callback: useCallback(
      (e: Triggers.UseEvent) => {
        const measureMode = determineMode(e.triggers);
        if (measureMode === "one" || measureMode === "two") onModeChange?.(measureMode);
      },
      [onModeChange],
    ),
  });

  const measureModeRef = useSyncedRef(mode);
  const hasSecondRef = useSyncedRef(state.two != null);
  const hasFirstRef = useSyncedRef(state.one != null);

  const handleClick: Viewport.UseHandler = useCallback(
    ({ mode, cursor }): void => {
      const measureMode = measureModeRef.current;
      if (mode === "click") {
        const isOne = measureMode === "one";
        const isTwo = measureMode === "two";
        setState((p) => ({ ...p, [measureMode]: cursor }));
        const moveToTwo = isOne && !hasSecondRef.current;
        const moveToOne = isTwo && !hasFirstRef.current;
        if (moveToTwo || moveToOne)
          // Add a small delay to allow the aether state to propagate. This ensures
          // that we don't accidentally clear the new point state we just set.
          setTimeout(() => onModeChange?.(moveToTwo ? "two" : "one"), 10);
      }
    },
    [setState, onModeChange],
  );

  LinePlot.useViewport(handleClick, "Measure.Measure");

  const handleMove = useCallback(
    (e: MouseEvent): void => setState((p) => ({ ...p, hover: xy.construct(e) })),
    [setState],
  );

  const handleLeave = useCallback((): void => {
    setState((p) => ({ ...p, hover: null }));
  }, [setState]);

  useEffect(() => {
    if (ref.current === null) return;
    // Select the parent node of the tooltip
    const parent = ref.current.parentElement;
    if (parent == null) return;
    // Bind a hover listener to the parent node
    parent.addEventListener("mousemove", handleMove);
    parent.addEventListener("mouseleave", handleLeave);
    return () => {
      parent.removeEventListener("mousemove", handleMove);
      parent.removeEventListener("mouseleave", handleLeave);
    };
  }, [handleClick]);

  return <span ref={ref} />;
};
