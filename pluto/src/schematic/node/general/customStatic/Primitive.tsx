// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type location } from "@synnaxlabs/x";
import { type ReactElement, useRef } from "react";

import { CSS } from "@/css";
import { Custom } from "@/schematic/node/common/custom";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";

export interface Props {
  specKey: string;
  orientation?: location.Outer;
  scale?: number;
  className?: string;
  stateOverrides?: schematic.symbol.State[];
}

export const Primitive = ({
  specKey,
  orientation = "left",
  scale = 1,
  className,
  stateOverrides,
}: Props): ReactElement => {
  const resolution = Custom.useResolveSymbol(specKey);
  const containerRef = useRef<HTMLDivElement>(null);
  Custom.useRender({
    container: containerRef.current,
    orientation,
    activeState: "base",
    externalScale: scale,
    spec: resolution.status === "resolved" ? resolution.spec : undefined,
    stateOverrides,
  });
  if (resolution.status === "missing")
    return <Custom.Missing orientation={orientation} className={className} />;
  const handles = resolution.status === "resolved" ? resolution.spec.handles : [];
  return (
    <Base.Div
      ref={containerRef}
      orientation={orientation}
      className={CSS(CSS.BM("symbol", "custom"), CSS.B("custom-static"), className)}
    >
      <Handle.Boundary orientation={orientation}>
        {handles.map((handle) => (
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
    </Base.Div>
  );
};
