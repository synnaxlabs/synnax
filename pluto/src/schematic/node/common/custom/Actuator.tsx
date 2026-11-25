// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import type { MouseEventHandler, ReactElement } from "react";

import { CSS } from "@/css";
import { Drawing } from "@/schematic/node/common/custom/Drawing";
import { usePaints } from "@/schematic/node/common/custom/paint";
import { isValidSVG } from "@/schematic/node/common/custom/render";
import type { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import { Warning } from "@/schematic/node/symbol/Warning";
import { Symbol } from "@/schematic/symbol";

export interface ActuatorProps
  extends
    Omit<Toggle.ButtonProps, "onClick">,
    Pick<Primitive.SVGBasedProps, "strokeColor"> {
  specKey: string;
  onClick?: MouseEventHandler<HTMLElement>;
  scale?: number;
  stateOverrides?: schematic.symbol.StateOverride[];
}

export const Actuator = ({
  specKey,
  enabled = false,
  triggered = false,
  orientation = "left",
  scale = 1,
  className,
  stateOverrides,
  strokeColor,
  ...rest
}: ActuatorProps): ReactElement => {
  const { symbol, missing, failed } = Symbol.useResolved(specKey);
  const spec = symbol?.data;
  const value = enabled ? 1 : 0;
  const paints = usePaints(spec, value, {
    override: stateOverrides?.[value],
    nodeColor: strokeColor,
  });
  const invalid = spec != null && !isValidSVG(spec.drawing.markup);
  if (missing || failed || invalid)
    return (
      <Warning
        cause={missing ? "missing" : failed ? "failed" : "invalid"}
        className={className}
      />
    );
  return (
    <Toggle.Button
      className={CSS.cls(
        CSS.BM("symbol", "custom"),
        CSS.B("custom-actuator"),
        orientation != null && CSS.loc(orientation),
        enabled && CSS.M("enabled"),
        triggered && CSS.M("triggered"),
        className,
      )}
      enabled={enabled}
      triggered={triggered}
      orientation={orientation}
      {...rest}
    >
      {spec != null && (
        <Drawing
          spec={spec}
          orientation={orientation}
          scale={scale}
          strokeColor={strokeColor}
          paints={paints}
        />
      )}
    </Toggle.Button>
  );
};
