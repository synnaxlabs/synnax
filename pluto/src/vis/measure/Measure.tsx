// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { xy } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";

import { Aether } from "@/aether";
import { Triggers } from "@/triggers";
import { type Viewport } from "@/viewport";
import { LinePlot } from "@/vis/lineplot";
import { measure } from "@/vis/measure/aether";

type ClickMode = "one" | "two" | "clear" | "empty";

const MEASURE_TRIGGERS: Triggers.ModeConfig<ClickMode> = {
  defaultMode: "empty",
  one: [["1"]],
  two: [["2"]],
  clear: [["Shift"]],
  empty: [[]],
};

const REDUCED_MEASURE_TRIGGERS = Triggers.flattenConfig(MEASURE_TRIGGERS);

 
export interface MeasureProps {}

export const Measure = Aether.wrap<MeasureProps>("Measure", ({ aetherKey }) => {
  const [, , setState] = Aether.use({
    aetherKey,
    type: measure.Measure.TYPE,
    schema: measure.measureStateZ,
    initialState: {
      hover: null,
      one: null,
      two: null,
    },
  });

  const ref = useRef<HTMLSpanElement>(null);

  const triggers = Triggers.useHeldRef({
    triggers: REDUCED_MEASURE_TRIGGERS,
    loose: true,
  });

  const handleClick: Viewport.UseHandler = useCallback(
    ({ mode, cursor }): void => {
      if (mode === "click") {
        const measureMode = Triggers.determineMode<ClickMode>(
          MEASURE_TRIGGERS,
          triggers.current.triggers,
          true,
        );
        if (["one", "two"].includes(measureMode))
          return setState((p) => ({ ...p, [measureMode]: cursor }));
        else if (measureMode === "clear")
          setState((p) => ({ ...p, one: null, two: null }));
        else
          setState((p) => ({
            ...p,
            one: p.one ?? cursor,
            two: p.one !== null ? cursor : p.two,
          }));
      }
    },
    [setState, triggers],
  );

  LinePlot.useViewport(handleClick);

  const handleMove = useCallback(
    (e: MouseEvent): void => setState((p) => ({ ...p, hover: xy.construct(e) })),
    [setState],
  );

  const handleLeave = useCallback((): void => {
    setState((p) => ({
      ...p,
      hover: null,
    }));
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
});
