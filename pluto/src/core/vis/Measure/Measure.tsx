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

import { AetherMeasure } from "./aether";

import { Aether } from "@/core/aether/main";

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

  const handleClick = useCallback(
    (e: MouseEvent): void => {
      setState((p) => ({
        ...p,
        one: p.one === null ? new XY(e) : p.one,
        two: p.one !== null ? new XY(e) : p.two,
      }));
    },
    [setState]
  );

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
    console.log("SOG");
    if (ref.current === null) return;
    // Select the parent node of the tooltip
    const parent = ref.current.parentElement;
    if (parent == null) return;
    // Bind a hover listener to the parent node
    parent.addEventListener("click", handleClick);
    parent.addEventListener("mousemove", handleMove);
    parent.addEventListener("mouseleave", handleLeave);
    return () => {
      parent.removeEventListener("click", handleClick);
      parent.removeEventListener("mousemove", handleMove);
    };
  }, [handleClick]);

  return <span ref={ref} />;
});
