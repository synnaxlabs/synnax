// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useRef } from "react";

import { XY } from "@synnaxlabs/x";

import { Aether } from "@/core/aether/main";
import { Triggers } from "@/core/triggers";
import { TriggerConfig } from "@/core/triggers/triggers";
import { useLinePlotViewport } from "@/core/vis/LinePlot/main/LinePlot";
import { AetherMeasure } from "@/core/vis/Measure/aether";
import { UseViewportHandler } from "@/core/vis/viewport";

type MeasureClickMode = "one" | "two" | "clear" | "empty";

const MEASURE_TRIGGERS: TriggerConfig<MeasureClickMode> = {
  defaultMode: "empty",
  one: [["1"]],
  two: [["2"]],
  clear: [["Shift"]],
  empty: [[]],
};

const REDUCED_MEASURE_TRIGGERS = Triggers.reduceConfig(MEASURE_TRIGGERS);

export interface MeasureProps {}

export const Measure = Aether.wrap<MeasureProps>("Measure", ({ aetherKey }) => {
  const [, , setState] = Aether.use({
    aetherKey,
    type: AetherMeasure.TYPE,
    schema: AetherMeasure.stateZ,
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

  const handleClick: UseViewportHandler = useCallback(
    ({ mode, cursor }): void => {
      if (mode === "click") {
        const measureMode = Triggers.determineMode<MeasureClickMode>(
          MEASURE_TRIGGERS,
          triggers.current.triggers,
          true
        );
        if (["one", "two"].includes(measureMode))
          return setState((p) => ({ ...p, [measureMode]: cursor }));
        else if (measureMode === "clear")
          setState((p) => ({ ...p, one: null, two: null }));
        else
          setState((p) => ({
            ...p,
            one: p.one === null ? cursor : p.one,
            two: p.one !== null ? cursor : p.two,
          }));
      }
    },
    [setState, triggers]
  );

  useLinePlotViewport(handleClick);

  const handleMove = useCallback(
    (e: MouseEvent): void => {
      setState((p) => ({
        ...p,
        hover: new XY(e),
      }));
    },
    [setState]
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
    };
  }, [handleClick]);

  return <span ref={ref} />;
});
