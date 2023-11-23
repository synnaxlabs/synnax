// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentPropsWithoutRef, type ReactElement } from "react";

import { dimensions, direction } from "@synnaxlabs/x";

import { Aether } from "@/aether";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Toggle } from "@/vis/toggle";

import "@/vis/valve/Valve.css";

export interface ValveProps
  extends Toggle.UseProps,
    Omit<ComponentPropsWithoutRef<"button">, "color"> {
  color?: Color.Crude;
  label?: string;
  direction?: direction.Direction;
}

const BASE_VALVE_DIMS: dimensions.Dimensions = {
  width: 106,
  height: 54,
};

export const Valve = Aether.wrap<ValveProps>(
  "Valve",
  ({
    aetherKey,
    color,
    style = {},
    className,
    source,
    sink,
    direction: dir = "x",
    ...props
  }): ReactElement => {
    const { triggered, enabled, toggle } = Toggle.use({ source, sink, aetherKey });
    const dir_ = direction.construct(dir);
    const dims = dir_ === "y" ? dimensions.swap(BASE_VALVE_DIMS) : BASE_VALVE_DIMS;
    // @ts-expect-error -- React css doesn't recognize variables
    if (color != null) style[CSS.var("base-color")] = new Color.Color(color).rgbString;
    return (
      <button
        className={CSS(
          className,
          CSS.B("valve"),
          triggered && CSS.BM("valve", "triggered"),
          enabled && CSS.BM("valve", "active"),
          CSS.dir(dir_),
        )}
        onClick={toggle}
        style={style}
        {...props}
      >
        <svg
          width={dims.width * 0.75}
          height={dims.height * 0.75}
          viewBox={dimensions.svgViewBox(dims)}
        >
          <path
            vectorEffect="non-scaling-stroke"
            d="M52 25.5L4.88003 2.41121C3.55123 1.7601 2 2.72744 2 4.20719V47.7349C2 49.2287 3.57798 50.1952 4.90865 49.5166L52 25.5ZM52 25.5L99.12 2.41121C100.449 1.7601 102 2.72744 102 4.2072V47.7349C102 49.2287 100.422 50.1952 99.0913 49.5166L52 25.5Z"
          />
        </svg>
      </button>
    );
  },
);
