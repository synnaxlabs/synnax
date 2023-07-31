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

import { useLinePlotViewport } from "../LinePlot/main/LinePlot";
import { UseViewportHandler } from "../viewport";

import { AetherMeasure } from "./aether";

import { Aether } from "@/core/aether/main";
import { Trigger, Triggers } from "@/core/triggers";

export interface MeasureProps {}

const ONE_TRIGGER: Trigger = ["1"];
const TWO_TRIGGER: Trigger = ["2"];

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
    triggers: [ONE_TRIGGER, TWO_TRIGGER],
    loose: true,
  });

  const handleClick: UseViewportHandler = useCallback(
    ({ mode, cursor }): void => {
      if (mode === "click") {
        if (Triggers.match([ONE_TRIGGER], triggers.current.triggers, true))
          setState((p) => ({ ...p, one: cursor }));
        else if (Triggers.match([TWO_TRIGGER], triggers.current.triggers, true))
          setState((p) => ({ ...p, two: cursor }));
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
