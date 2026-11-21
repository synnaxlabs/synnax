// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useSyncedRef, useUniqueKey } from "@synnaxlabs/lyra/hooks";
import { box, xy } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef } from "react";
import type { z } from "zod";

import { Aether } from "@/aether";
import { tooltip } from "@/lineplot/tooltip/aether";

export interface TooltipProps
  extends
    Omit<z.input<typeof tooltip.tooltipStateZ>, "position" | "hovered">,
    Aether.ComponentProps {
  /**
   * Called with the line nearest the cursor, or null when no line is near or the
   * tooltip unmounts.
   */
  onHoveredChange?: (hovered: tooltip.Hovered | null) => void;
}

export const Tooltip = ({
  aetherKey,
  onHoveredChange,
  ...rest
}: TooltipProps): ReactElement | null => {
  const cKey = useUniqueKey(aetherKey);
  const reported = useRef(false);
  const onHoveredChangeRef = useSyncedRef(onHoveredChange);
  // A lifecycle with no React state, so the cursor moving does not re-render.
  const { setState } = Aether.useLifecycle({
    aetherKey: cKey,
    type: tooltip.Tooltip.TYPE,
    schema: tooltip.tooltipStateZ,
    initialState: { position: null, hovered: null, ...rest },
    onAetherChange: ({ hovered }) => {
      reported.current = hovered != null;
      onHoveredChange?.(hovered);
    },
  });
  useEffect(
    () => () => {
      if (reported.current) onHoveredChangeRef.current?.(null);
    },
    [onHoveredChangeRef],
  );
  const { relative } = rest;
  useEffect(() => setState((p) => ({ ...p, relative })), [relative]);

  const ref = useRef<HTMLSpanElement>(null);

  const handleMove = (e: MouseEvent): void => {
    const canvas = document.querySelector(".pluto-canvas-container");
    if (canvas == null) return;
    const topLeft = box.topLeft(canvas);
    setState((p) => ({ ...p, position: xy.sub(xy.construct(e), topLeft) }));
  };

  const handleLeave = (): void => setState((p) => ({ ...p, position: null }));

  useEffect(() => {
    if (ref.current === null) return;
    const parent = ref.current.parentElement;
    if (parent == null) return;
    parent.addEventListener("mousemove", handleMove);
    parent.addEventListener("mouseleave", handleLeave);
    return () => {
      parent.removeEventListener("mousemove", handleMove);
      parent.removeEventListener("mouseleave", handleLeave);
    };
  }, [handleMove]);

  return <span ref={ref} />;
};
