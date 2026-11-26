// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { useCombinedRefs } from "@synnaxlabs/lyra/hooks";
import type { CrudeTimeSpan } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { CSS } from "@/css";
import { usePartClicks } from "@/schematic/node/common/custom/clicks";
import { moves, type Reading } from "@/schematic/node/common/custom/motion";
import { useMotionContext } from "@/schematic/node/common/custom/MotionProvider";
import type { Paint } from "@/schematic/node/common/custom/paint";
import { useRender } from "@/schematic/node/common/custom/render";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { Region } from "@/schematic/node/common/region";

export interface DrawingProps extends Primitive.SVGBasedProps {
  spec: schematic.symbol.Spec;
  /**
   * Marks a drawing that ships with Pluto. Its markup mounts without the sanitizer.
   * Shapes with no stroke of their own take the symbol color, so the hover and pressed
   * colors of a toggle reach them.
   */
  builtIn?: boolean;
  /** paints are the paints of the spec's looks, from usePaints. */
  paints?: Paint[];
  /**
   * readings holds the reading of each state of the spec by key. Copy it on change.
   * A state with no reading reads as a state with no source.
   */
  readings?: Record<string, Reading>;
  /**
   * motionDisabled stops spin, flow, and blink, and makes level, rotate, and move jump
   * to each new fraction.
   */
  motionDisabled?: boolean;
  /** partClicks holds what a click on each part does, by part key. */
  partClicks?: Record<string, () => void>;
  /** onClickDelay is how long a press on a part lasts before its click runs. */
  onClickDelay?: CrudeTimeSpan;
}

const NO_READINGS: Record<string, Reading> = {};
const NO_CLICKS: Record<string, () => void> = {};

/**
 * Draws a custom symbol and its handles from a spec, inside a Primitive.Div or a
 * Toggle.Button. It draws nothing when the markup does not parse. A spec whose looks
 * move needs a MotionProvider above the drawing.
 */
export const Drawing = ({
  spec,
  paints,
  builtIn = false,
  orientation = "left",
  scale = 1,
  strokeColor,
  readings = NO_READINGS,
  motionDisabled = false,
  partClicks = NO_CLICKS,
  onClickDelay,
}: DrawingProps): ReactElement | null => {
  const context = useMotionContext();
  if (context == null && spec.looks.some(moves))
    throw new Error("Custom.Drawing draws motion only inside a Custom.MotionProvider");
  const motion =
    context == null ? undefined : { ...context, readings, disabled: motionDisabled };
  const setDrawing = useRender({ spec, paints, strokeColor, builtIn, motion });
  const { ref, ...parts } = usePartClicks({
    spec,
    clicks: partClicks,
    delay: onClickDelay,
  });
  const hostRef = useCombinedRefs(setDrawing, ref);
  const dims = Region.measure(spec.drawing.markup);
  if (dims == null) return null;
  // Primitive.SVG shrinks built-in drawings by BASE_SCALE. A spec sets its own scale.
  const svgScale =
    (Primitive.resolveScale(spec.scale) * Primitive.resolveScale(scale)) /
    Primitive.BASE_SCALE;
  return (
    <>
      <Primitive.SVG
        dimensions={dims}
        orientation={orientation}
        scale={svgScale}
        strokeColor={strokeColor}
        className={CSS.cls(
          CSS.BE("symbol", "drawing"),
          !builtIn && CSS.B("custom-drawing"),
        )}
      >
        <g ref={hostRef} {...parts} />
      </Primitive.SVG>
      <Handle.Boundary orientation={orientation}>
        {spec.handles.map((handle) => (
          <Handle.Handle
            key={handle.key}
            id={handle.key}
            location={handle.orientation}
            orientation={orientation}
            left={handle.position.x * 100}
            top={handle.position.y * 100}
          />
        ))}
      </Handle.Boundary>
    </>
  );
};
