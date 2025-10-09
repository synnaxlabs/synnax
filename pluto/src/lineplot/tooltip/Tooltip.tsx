// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useRef } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { tooltip } from "@/lineplot/tooltip/aether";

export interface TooltipProps
  extends Omit<z.input<typeof tooltip.tooltipStateZ>, "position">,
    Aether.ComponentProps {}

export const Tooltip = ({ aetherKey, ...rest }: TooltipProps): ReactElement | null => {
  const cKey = useUniqueKey(aetherKey);
  const [, , setState] = Aether.use({
    aetherKey: cKey,
    type: tooltip.Tooltip.TYPE,
    schema: tooltip.tooltipStateZ,
    initialState: { position: null, ...rest },
  });

  const ref = useRef<HTMLSpanElement>(null);

  const handleMove = useCallback(
    (e: MouseEvent): void => {
      // select the .pluto-canvas-container element
      const canvas = document.querySelector(".pluto-canvas-container");
      if (canvas == null) return;
      const topLeft = box.topLeft(canvas);
      setState({ position: xy.translation(topLeft, xy.construct(e)) });
    },
    [setState],
  );

  const handleLeave = useCallback((): void => setState({ position: null }), [setState]);

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
  }, [handleMove]);

  return <span ref={ref} />;
};
