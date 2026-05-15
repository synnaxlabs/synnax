// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x/location";
import { CSS } from "@synnaxlabs/charon/css";
import { type schematic } from "@synnaxlabs/client";

import { type ReactElement, useRef, useState } from "react";

import { Custom } from "@/schematic/node/common/custom";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
import { useRetrieveEffect } from "@/schematic/symbol/queries";

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
  const [spec, setSpec] = useState<schematic.symbol.Spec | undefined>(undefined);
  useRetrieveEffect({
    query: { key: specKey },
    onChange: (res) => setSpec(res.data?.data),
  });
  const containerRef = useRef<HTMLDivElement>(null);
  Custom.useRender({
    container: containerRef.current,
    orientation,
    activeState: "base",
    externalScale: scale,
    spec,
    stateOverrides,
  });
  const handles = spec?.handles ?? [];
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
