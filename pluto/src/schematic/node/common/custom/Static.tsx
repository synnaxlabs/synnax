// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import type { location } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { CSS } from "@/css";
import { Drawing } from "@/schematic/node/common/custom/Drawing";
import { usePaints } from "@/schematic/node/common/custom/paint";
import { isValidSVG } from "@/schematic/node/common/custom/render";
import { Primitive } from "@/schematic/node/common/primitive";
import { Warning } from "@/schematic/node/symbol/Warning";
import { Symbol } from "@/schematic/symbol";

export interface StaticProps {
  specKey: string;
  orientation?: location.Outer;
  scale?: number;
  className?: string;
  stateOverrides?: schematic.symbol.StateOverride[];
}

export const Static = ({
  specKey,
  orientation = "left",
  scale = 1,
  className,
  stateOverrides,
}: StaticProps): ReactElement => {
  const { symbol, missing, failed } = Symbol.useResolved(specKey);
  const spec = symbol?.data;
  const paints = usePaints(spec, 0, { override: stateOverrides?.[0] });
  const invalid = spec != null && !isValidSVG(spec.drawing.markup);
  if (missing || failed || invalid)
    return (
      <Warning
        cause={missing ? "missing" : failed ? "failed" : "invalid"}
        className={className}
      />
    );
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS.cls(CSS.BM("symbol", "custom"), CSS.B("custom-static"), className)}
    >
      {spec != null && (
        <Drawing spec={spec} orientation={orientation} scale={scale} paints={paints} />
      )}
    </Primitive.Div>
  );
};
