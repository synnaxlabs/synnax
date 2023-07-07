// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useRef, useState } from "react";

import { set, z } from "zod";

import { AetherRule } from "./aether";

import { Aether } from "@/core/aether/main";
import { useCursorDrag } from "@/core/hooks/useCursorDrag";
import { Box, XY } from "@synnaxlabs/x";
import { preventDefault } from "@/util/event";
import { Text } from "@/core/std";

export interface RuleProps extends Omit<z.input<typeof AetherRule.stateZ>, "dragging"> {
  label: string;
}

export const Rule = Aether.wrap<RuleProps>(
  "Rule",
  ({ aetherKey,label,  ...props }): ReactElement | null => {
    const [,{position, pixelPosition}, setState] = Aether.use({
      aetherKey,
      type: AetherRule.TYPE,
      schema: AetherRule.stateZ,
      initialState: {
        ...props,
        dragging: false,
      },
    });

    const pixelPosRef = useRef(pixelPosition)


    const handleDragStart = useCursorDrag({
      onStart: useCallback((loc:XY) => {
        setState((p) => ({...p, dragging: true}))
        pixelPosRef.current = loc.y;
      }, []),
      onMove: (box: Box) => {
        setState((p) => ({...p, pixelPosition:  pixelPosRef.current + box.signedHeight }))
      },
      onEnd: useCallback((box: Box) => {
        setState((p) => ({...p, dragging: false}))
        pixelPosRef.current = pixelPosition
      }, []),
    })

    return (<div style={{
      height: "1rem", 
      position: "absolute",
      top: `calc(${pixelPosition}px - 0.5rem)`,
      width: "auto",
      // overflow: "hidden",
      cursor: "ns-resize",
      gridColumnStart: "plot-start",
      gridColumnEnd: "plot-end"
    }} 
    draggable
      onDrag={preventDefault}
      onDragEnd={preventDefault}
      onDragStart={handleDragStart}
      >
        <Text level="p" style={{position: "relative", left: 80, bottom: "2rem"}}>{`${label} ${position.toFixed(2)} psi`}</Text>
      </div>
    )
  }
);
